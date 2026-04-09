package handler

import "github.com/elecbug/linuxus/src/auth/internal/page"

func (a *App) GetLoginPage() string {
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
				"action", "/"+a.loginPath,
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
		page.NewHTML(
			"footer",
			page.NewAttributes(),
			"© 2026 ",
			page.NewHTML(
				"a",
				page.NewAttributes("href", "https://github.com/elecbug/linuxus"),
				"Linuxus",
			),
			". All rights reserved.",
		),
	)

	return htmlpage.Render()
}

func (a *App) GetServicePage() string {
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
				"linuxus | {{.ID}}",
			),
			page.NewHTML(
				"div",
				page.NewAttributes("class", "right"),
				page.NewHTML(
					"a",
					page.NewAttributes(
						"class", "btn",
						"href", "/"+a.terminalPath+"/",
						"target", "shellframe",
					),
					"Open Shell",
				),
				page.NewHTML(
					"a",
					page.NewAttributes(
						"class", "btn btn-danger",
						"href", "/"+a.logoutPath,
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
				page.NewAttributes("name", "shellframe", "src", "/"+a.terminalPath+"/"),
				"", // The iframe content will be loaded from the terminal path
			),
		),
	)

	return htmlpage.Render()
}

func (a *App) GetErrorPage() string {
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
		page.NewHTML(
			"footer",
			page.NewAttributes(),
			"© 2026 ",
			page.NewHTML(
				"a",
				page.NewAttributes("href", "https://github.com/elecbug/linuxus"),
				"Linuxus",
			),
			". All rights reserved.",
		),
	)

	return htmlpage.Render()

}

func getBaseMeta() []page.Attribute {
	return page.NewAttributes(
		"charset", "UTF-8",
		"name", "viewport",
		"content", "width=device-width, initial-scale=1.0",
	)
}
