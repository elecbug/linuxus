package page

import (
	"github.com/elecbug/linuxus/src/auth/internal/html"
)

func GetLoginPage(loginPath string) string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | Login",
		getBaseMeta(),
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
		html.NewHTML(
			"p",
			html.NewAttributes("class", "tooltip"),
			"Don't have an account? Contact the administrator.",
		),
		linuxusFooterHTML(),
	)

	return htmlpage.Render()
}

func GetServicePage(terminalPath, logoutPath string) string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | {{.ID}}",
		getBaseMeta(),
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

func GetErrorPage() string {
	htmlpage := html.NewHTMLPage(
		"Linuxus | Error",
		getBaseMeta(),
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

func getBaseMeta() []html.Attribute {
	return html.NewAttributes(
		"charset", "UTF-8",
		"name", "viewport",
		"content", "width=device-width, initial-scale=1.0",
	)
}

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
