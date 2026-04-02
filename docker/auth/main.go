package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const LOGIN_PATH = "login"
const LOGOUT_PATH = "logout"
const SERVICE_PATH = "service"

type App struct {
	users      map[string]string
	sessionKey []byte
	loginTmpl  *template.Template
}

func main() {
	studentsFile := getEnv("STUDENTS_FILE", "/data/students.txt")
	sessionSecret := getEnv("SESSION_SECRET", "replace-this-with-a-long-random-secret-key")

	users, err := loadUsers(studentsFile)
	if err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	tmpl, err := template.New(LOGIN_PATH).Parse(loginPage)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	app := &App{
		users:      users,
		sessionKey: []byte(sessionSecret),
		loginTmpl:  tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleRoot)
	mux.HandleFunc("/"+LOGIN_PATH, app.handleLogin)
	mux.HandleFunc("/"+LOGOUT_PATH, app.handleLogout)

	// Important:
	// /$SERVICE_PATH  -> redirect to /$SERVICE_PATH/
	// /$SERVICE_PATH/ -> reverse proxy with prefix stripping
	mux.HandleFunc("/"+SERVICE_PATH, app.handleShellRedirect)
	mux.HandleFunc("/"+SERVICE_PATH+"/", app.handleShellProxy)

	addr := ":8080"
	log.Printf("Auth server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func loadUsers(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	users := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in students file: %s", line)
		}

		studentID := strings.TrimSpace(parts[0])
		hash := strings.TrimSpace(parts[1])

		if studentID == "" || hash == "" {
			return nil, fmt.Errorf("invalid line in students file: %s", line)
		}

		users[studentID] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionStudentID(r); ok {
		http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.renderLogin(w, "")
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		studentID := strings.TrimSpace(r.FormValue("student_id"))
		password := r.FormValue("password")

		hash, ok := a.users[studentID]
		if !ok {
			a.renderLogin(w, "Invalid student ID or password")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			a.renderLogin(w, "Invalid student ID or password")
			return
		}

		a.setSessionCookie(w, studentID)
		http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
}

func (a *App) handleShellRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionStudentID(r); !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
}

func (a *App) handleShellProxy(w http.ResponseWriter, r *http.Request) {
	studentID, ok := a.getSessionStudentID(r)
	if !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	safeID := sanitizeStudentID(studentID)
	targetURL := fmt.Sprintf("http://student_%s:7681", safeID)

	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid backend target", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Strip "$SERVICE_PATH" prefix before forwarding to ttyd.
		// Examples:
		//   /$SERVICE_PATH/      -> /
		//   /$SERVICE_PATH/ws    -> /ws
		//   /$SERVICE_PATH/foo   -> /foo
		newPath := strings.TrimPrefix(req.URL.Path, "/"+SERVICE_PATH)
		if newPath == "" {
			newPath = "/"
		}
		if !strings.HasPrefix(newPath, "/") {
			newPath = "/" + newPath
		}

		req.URL.Path = newPath
		req.URL.RawPath = ""

		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", studentID, err)
		http.Error(w, "Service backend is unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

func (a *App) renderLogin(w http.ResponseWriter, errMsg string) {
	data := struct {
		Error string
	}{
		Error: errMsg,
	}

	if err := a.loginTmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func sanitizeStudentID(id string) string {
	id = strings.ToLower(id)
	var b strings.Builder

	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' || ch == '.' {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}

	result := b.String()
	result = strings.TrimLeft(result, "._-")
	if result == "" {
		return "invalid"
	}
	return result
}

func (a *App) setSessionCookie(w http.ResponseWriter, studentID string) {
	signature := a.sign(studentID)
	payload := studentID + "|" + signature
	value := base64.StdEncoding.EncodeToString([]byte(payload))

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(12 * time.Hour),
	})
}

func (a *App) getSessionStudentID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", false
	}

	raw, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", false
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", false
	}

	studentID := parts[0]
	signature := parts[1]

	expected := a.sign(studentID)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", false
	}

	return studentID, true
}

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	mac.Write([]byte(value))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

const loginPage = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Linuxus Login</title>
    <style>
        body {
            font-family: sans-serif;
            max-width: 420px;
            margin: 60px auto;
        }
        form {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        input {
            padding: 10px;
            font-size: 16px;
        }
        button {
            padding: 10px;
            font-size: 16px;
        }
        .error {
            color: red;
        }
        .links {
            margin-top: 16px;
            display: flex;
            gap: 12px;
        }
    </style>
</head>
<body>
    <h2>Linuxus Login</h2>
    {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
    <form method="post" action="/` + LOGIN_PATH + `">
        <input type="text" name="student_id" placeholder="Student ID" required>
        <input type="password" name="password" placeholder="Password" required>
        <button type="submit">Login</button>
    </form>
    <div class="links">
        <a href="/` + SERVICE_PATH + `/">Go to service</a>
        <a href="/` + LOGOUT_PATH + `">Logout</a>
    </div>
</body>
</html>
`
