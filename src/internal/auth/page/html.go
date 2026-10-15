// Package page contains the shared authentication and application templates.
package page

import (
	"embed"
	"html/template"
)

//go:embed *.html
var templates embed.FS

// Data supplies configured routes and session details to the shared interface.
type Data struct {
	Title, Error, ID, View                                            string
	LoginURL, SignupURL, ServiceURL, TerminalURL, LogoutURL, AdminURL string
	BaseURL, APIURL, Token, AdminID                                   string
	IsAdmin, AllowSignup                                              bool
}

// NewTemplates loads the shared layouts. Callers add the administrator panel.
func NewTemplates() (*template.Template, error) {
	return template.New("linuxus").ParseFS(templates, "*.html")
}
