package handler

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/elecbug/linuxus/src/auth/internal/page"
	"github.com/elecbug/linuxus/src/auth/internal/user"
)

// handleSignup processes GET and POST requests to the signup endpoint for user registration.
func (a *App) handleSignup(w http.ResponseWriter, r *http.Request) {
	if !a.allowSignup {
		a.renderError(w, "User signup is currently disabled.", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.renderSignup(w, "")
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			a.renderError(w, "Bad request", http.StatusBadRequest)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		if id == "" || password == "" {
			a.renderSignup(w, "ID and password are required.")
			return
		}

		if password != confirmPassword {
			a.renderSignup(w, "Passwords do not match.")
			return
		}

		if _, exists := a.users[id]; exists {
			a.renderSignup(w, "This ID is already registered.")
			return
		}

		if err := user.AddUser(a.authListFile, a.users, id, password); err != nil {
			a.renderSignup(w, "Failed to create user.")
			return
		}

		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return

	default:
		a.renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// renderSignup renders the signup page with an optional error message.
func (a *App) renderSignup(w http.ResponseWriter, errMsg string) {
	tmpl, err := template.New("signup").Parse(
		page.GetSignupPage(a.signupPath, a.loginPath),
	)
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Error string
	}{
		Error: errMsg,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}
