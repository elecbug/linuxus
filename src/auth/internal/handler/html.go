package handler

import "github.com/elecbug/linuxus/src/auth/internal/page"

func (a *App) GetLoginPage() string {
	htmlpage := page.NewHTMLPage(
		"Linuxus | Login",
		getBaseMeta(),
		getLoginCSS(),
		page.NewHTML("h2", []page.KeyValue{}, "Linuxus Login"),
		page.NewHTML("p", []page.KeyValue{{Key: "class", Value: "error"}}, "{{.Error}}").AddPrefix("{{if .Error}}").AddSuffix("{{end}}"),
		page.NewHTML("form", []page.KeyValue{{Key: "method", Value: "post"}, {Key: "action", Value: "/" + a.loginPath}},
			page.NewHTML("input", []page.KeyValue{{Key: "type", Value: "text"}, {Key: "name", Value: "id"}, {Key: "placeholder", Value: "ID"}, {Key: "required", Value: "true"}}),
			page.NewHTML("input", []page.KeyValue{{Key: "type", Value: "password"}, {Key: "name", Value: "password"}, {Key: "placeholder", Value: "Password"}, {Key: "required", Value: "true"}}),
			page.NewHTML("button", []page.KeyValue{{Key: "type", Value: "submit"}}, "Login"),
		),
	)

	return htmlpage.Render()
}

func (a *App) GetServicePage() string {
	htmlpage := page.NewHTMLPage(
		"Linuxus | {{.ID}}",
		getBaseMeta(),
		getServiceCSS(),
		page.NewHTML("div", []page.KeyValue{{Key: "class", Value: "topbar"}},
			page.NewHTML("div", []page.KeyValue{{Key: "class", Value: "left"}}, "linuxus | {{.ID}}"),
			page.NewHTML("div", []page.KeyValue{{Key: "class", Value: "right"}},
				page.NewHTML("a", []page.KeyValue{{Key: "class", Value: "btn"}, {Key: "href", Value: "/" + a.terminalPath + "/"}, {Key: "target", Value: "shellframe"}}, "Open Shell"),
				page.NewHTML("a", []page.KeyValue{{Key: "class", Value: "btn btn-danger"}, {Key: "href", Value: "/" + a.logoutPath}}, "Logout"),
			),
		),
		page.NewHTML("div", []page.KeyValue{{Key: "class", Value: "frame-wrap"}},
			page.NewHTML("iframe", []page.KeyValue{{Key: "name", Value: "shellframe"}, {Key: "src", Value: "/" + a.terminalPath + "/"}}, ""),
		),
	)

	return htmlpage.Render()
}

func getBaseMeta() []page.KeyValue {
	return []page.KeyValue{
		{Key: "charset", Value: "UTF-8"},
	}
}
