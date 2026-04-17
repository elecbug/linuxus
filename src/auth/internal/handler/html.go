package handler

import (
	"github.com/elecbug/linuxus/src/auth/internal/page"
)

// getLoginPage renders the login page template source.
func getLoginPage(app *App) string {
	htmlpage := page.NewHTMLPage(
		"Linuxus | Login",
		getBaseMeta(),
		getLoginCSS(),
		page.NewHTML(
			"h2",
			page.NewAttributes(),
			"Linuxus Login",
		),
		page.NewHTML(
			"p",
			page.NewAttributes("class", "error"),
			"{{.Error}}",
		).AddPrefix("{{if .Error}}").AddSuffix("{{end}}"),
		page.NewHTML(
			"form",
			page.NewAttributes(
				"class", "login-form",
				"method", "post",
				"action", "/"+app.LoginPath(),
			),
			page.NewHTML(
				"input",
				page.NewAttributes(
					"type", "text",
					"name", "id",
					"placeholder", "ID",
					"required", "true",
				),
			),
			page.NewHTML(
				"input",
				page.NewAttributes(
					"type", "password",
					"name", "password",
					"placeholder", "Password",
					"required", "true",
				),
			),
			page.NewHTML(
				"button",
				page.NewAttributes("type", "submit"),
				"Login",
			),
		),
		page.NewHTML(
			"p",
			page.NewAttributes("class", "tooltip"),
			"Don't have an account? Contact the administrator.",
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// getServicePage renders the authenticated service page template source.
func getServicePage(app *App) string {
	htmlpage := page.NewHTMLPage(
		"Linuxus | {{.ID}}",
		getBaseMeta(),
		getServiceCSS(),
		page.NewHTML(
			"div",
			page.NewAttributes("class", "topbar"),
			page.NewHTML(
				"div",
				page.NewAttributes("class", "left"),
				page.NewHTML("p",
					page.NewAttributes(),
					"Linuxus | {{.ID}}",
				),
			),
			page.NewHTML(
				"div",
				page.NewAttributes("class", "right"),
				page.NewHTML(
					"a",
					page.NewAttributes(
						"class", "btn",
						"href", "/"+app.TerminalPath()+"/",
						"target", "shellframe",
					),
					"Open Shell",
				),
				page.NewHTML(
					"a",
					page.NewAttributes(
						"class", "btn btn-danger",
						"href", "/"+app.LogoutPath(),
					),
					"Logout",
				),
			),
		),
		page.NewHTML(
			"div",
			page.NewAttributes("class", "frame-wrap"),
			page.NewHTML(
				"iframe",
				page.NewAttributes(
					"name", "shellframe",
					"src", "/"+app.TerminalPath()+"/",
				),
				"", // The iframe content will be loaded from the terminal path
			),
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// getErrorPage renders the generic error page template source.
func getErrorPage() string {
	htmlpage := page.NewHTMLPage(
		"Linuxus | Error",
		getBaseMeta(),
		getErrorCSS(),
		page.NewHTML(
			"h2",
			page.NewAttributes(),
			"An Error Occurred",
		),
		page.NewHTML(
			"p",
			page.NewAttributes("class", "error"),
			"{{.Error}}",
		),
		page.NewHTML(
			"p",
			page.NewAttributes("class", "tooltip"),
			"Please try again or contact the administrator.",
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

// getBaseMeta returns default meta tag attributes for rendered pages.
func getBaseMeta() []page.Attribute {
	return page.NewAttributes(
		"charset", "UTF-8",
		"name", "viewport",
		"content", "width=device-width, initial-scale=1.0",
	)
}

// linuxusFooterHTML returns a shared footer HTML block.
func linuxusFooterHTML() *page.HTML {
	return page.NewHTML(
		"footer",
		page.NewAttributes(),
		"© 2026 ",
		page.NewHTML(
			"a",
			page.NewAttributes("href", "https://github.com/elecbug/linuxus"),
			"Linuxus",
		),
		". All rights reserved.",
	)
}
