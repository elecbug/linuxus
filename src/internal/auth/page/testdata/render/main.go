// Render the production templates with fixed, non-sensitive browser test data.
package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"

	"github.com/elecbug/linuxus/src/internal/auth/page"
)

func main() {
	templates := template.Must(template.Must(page.NewTemplates()).ParseFS(os.DirFS("internal/auth/handler"), "admin.html"))
	data := page.Data{
		ID: "admin", AdminID: "admin", IsAdmin: true, AllowSignup: true,
		LoginURL: "/auth/login", SignupURL: "/auth/signup", LogoutURL: "/auth/logout",
		ServiceURL: "/workspace/", TerminalURL: "/shell/", AdminURL: "/ops/classroom",
		BaseURL: "/ops/classroom", APIURL: "/ops/classroom/api/users", Token: "initial-token",
	}
	rendered := map[string]string{}
	for _, name := range []string{"admin", "workspace", "user", "login", "signup", "error"} {
		value := data
		templateName := name
		switch name {
		case "admin":
			templateName, value.Title, value.View = "app", "Administration", "admin"
		case "workspace", "user":
			templateName, value.Title, value.View = "app", "Workspace", "workspace"
			if name == "user" {
				value.ID, value.IsAdmin, value.Token = "alice", false, ""
			}
		case "login":
			value.Title = "Sign in"
		case "signup":
			value.Title = "Create account"
		case "error":
			value.Title, value.Error = "Unable to continue", "The environment is currently unavailable."
		}
		var out bytes.Buffer
		if err := templates.ExecuteTemplate(&out, templateName, value); err != nil {
			panic(err)
		}
		rendered[name] = out.String()
	}
	if err := json.NewEncoder(os.Stdout).Encode(rendered); err != nil {
		panic(err)
	}
}
