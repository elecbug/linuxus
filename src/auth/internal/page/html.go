package page

import (
	"github.com/elecbug/linuxus/src/auth/internal/html"
)

// GetLoginPage renders the login HTML template.
func GetLoginPage(loginPath string, allowSignup bool, signupPath string) string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | Login",
		getBaseMeta(),
		getFaviconLink(),
		getLoginCSS(),
		html.NewHTML(
			"h2",
			html.NewAttributes(),
			"Linuxus Login",
		),
		html.NewHTML(
			"p",
			html.NewAttributes("class", "error"),
			"{{.Error}}",
		).AddPrefix("{{if .Error}}").AddSuffix("{{end}}"),
		html.NewHTML(
			"form",
			html.NewAttributes(
				"class", "login-form",
				"method", "post",
				"action", "/"+loginPath,
			),
			html.NewHTML(
				"input",
				html.NewAttributes(
					"type", "text",
					"name", "id",
					"placeholder", "ID",
					"required", "true",
				),
			),
			html.NewHTML(
				"input",
				html.NewAttributes(
					"type", "password",
					"name", "password",
					"placeholder", "Password",
					"required", "true",
				),
			),
			html.NewHTML(
				"button",
				html.NewAttributes("type", "submit"),
				"Login",
			),
		),
		getAllowSignupHTML(allowSignup, signupPath),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

func getAllowSignupHTML(allowSignup bool, signupPath string) *html.HTML {
	if allowSignup {
		return html.NewHTML(
			"p",
			html.NewAttributes(
				"class", "tooltip",
			),
			html.NewHTML(
				"a",
				html.NewAttributes(
					"class", "signup-link",
					"href", "/"+signupPath,
				),
				"Don't have an account? Sign up here.",
			),
		)
	} else {
		return html.NewHTML(
			"p",
			html.NewAttributes(
				"class", "tooltip",
			),
			"New user signups are currently disabled.",
		)
	}
}

// GetServicePage renders the post-login service HTML template.
func GetServicePage(terminalPath, logoutPath string) string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | {{.ID}}",
		getBaseMeta(),
		getFaviconLink(),
		getServiceCSS(),
		html.NewHTML(
			"div",
			html.NewAttributes("class", "topbar"),
			html.NewHTML(
				"div",
				html.NewAttributes("class", "left"),
				html.NewHTML("p",
					html.NewAttributes(),
					"Linuxus | {{.ID}}",
				),
			),
			html.NewHTML(
				"div",
				html.NewAttributes("class", "right"),
				html.NewHTML(
					"a",
					html.NewAttributes(
						"class", "btn",
						"href", "/"+terminalPath+"/",
						"target", "shellframe",
					),
					"Open Shell",
				),
				html.NewHTML(
					"a",
					html.NewAttributes(
						"class", "btn btn-danger",
						"href", "/"+logoutPath,
					),
					"Logout",
				),
			),
		),
		html.NewHTML(
			"div",
			html.NewAttributes("class", "frame-wrap"),
			html.NewHTML(
				"iframe",
				html.NewAttributes(
					"name", "shellframe",
					"src", "/"+terminalPath+"/",
				),
				"", // The iframe content will be loaded from the terminal path
			),
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// GetErrorPage renders a generic error HTML template.
func GetErrorPage() string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | Error",
		getBaseMeta(),
		getFaviconLink(),
		getErrorCSS(),
		html.NewHTML(
			"h2",
			html.NewAttributes(),
			"An Error Occurred",
		),
		html.NewHTML(
			"p",
			html.NewAttributes("class", "error"),
			"{{.Error}}",
		),
		html.NewHTML(
			"p",
			html.NewAttributes("class", "tooltip"),
			"Please try again or contact the administrator.",
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// GetSignupPage renders the user registration HTML template.
func GetSignupPage(signupPath, loginPath string) string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | Sign Up",
		getBaseMeta(),
		getFaviconLink(),
		getLoginCSS(),
		html.NewHTML(
			"h2",
			html.NewAttributes(),
			"Linuxus Sign Up",
		),
		html.NewHTML(
			"p",
			html.NewAttributes("class", "error"),
			"{{.Error}}",
		).AddPrefix("{{if .Error}}").AddSuffix("{{end}}"),
		html.NewHTML(
			"form",
			html.NewAttributes(
				"class", "login-form",
				"method", "post",
				"action", "/"+signupPath,
			),
			html.NewHTML(
				"input",
				html.NewAttributes(
					"type", "text",
					"name", "id",
					"placeholder", "ID",
					"required", "true",
				),
			),
			html.NewHTML(
				"input",
				html.NewAttributes(
					"type", "password",
					"name", "password",
					"placeholder", "Password",
					"required", "true",
				),
			),
			html.NewHTML(
				"input",
				html.NewAttributes(
					"type", "password",
					"name", "confirm_password",
					"placeholder", "Confirm Password",
					"required", "true",
				),
			),
			html.NewHTML(
				"button",
				html.NewAttributes("type", "submit"),
				"Sign Up",
			),
		),
		html.NewHTML(
			"p",
			html.NewAttributes("class", "tooltip"),
			html.NewHTML(
				"a",
				html.NewAttributes(
					"class", "signup-link",
					"href", "/"+loginPath,
				),
				"Already have an account? Login here.",
			),
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// getBaseMeta returns base head meta entries used by all pages.
func getBaseMeta() []html.Attribute {
	return html.NewAttributes(
		"charset", "UTF-8",
		"name", "viewport",
		"content", "width=device-width, initial-scale=1.0",
	)
}

// getFaviconLink returns the link tag for the favicon.
func getFaviconLink() map[string][]html.Attribute {
	return map[string][]html.Attribute{
		"icon": html.NewAttributes(
			"rel", "icon",
			"type", "image/png",
			"href", "/static/favicon.png",
		),
	}
}

// linuxusFooterHTML builds the shared footer element.
func linuxusFooterHTML() *html.HTML {
	return html.NewHTML(
		"footer",
		html.NewAttributes(),
		"© 2026 ",
		html.NewHTML(
			"a",
			html.NewAttributes("href", "https://github.com/elecbug/linuxus"),
			"Linuxus",
		),
		". All rights reserved.",
	)
}
