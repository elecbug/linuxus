package handler

import "github.com/elecbug/linuxus/src/auth/internal/page"

func (a *App) GetLoginPage() string {
	// 	return `
	// <!DOCTYPE html>
	// <html>
	// <head>
	//     <meta charset="UTF-8">
	//     <title>Linuxus Login</title>
	//     ` + a.GetLoginCSS() + `
	// </head>
	// <body>
	//     <h2>Linuxus Login</h2>
	//     {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
	//     <form method="post" action="/` + a.loginPath + `">
	//         <input type="text" name="id" placeholder="ID" required>
	//         <input type="password" name="password" placeholder="Password" required>
	//         <button type="submit">Login</button>
	//     </form>
	// </body>
	// </html>
	// `

	htmlpage := page.NewHTMLPage(
		"Linuxus Login",
		[]page.KeyValue{
			{Key: "charset", Value: "UTF-8"},
		},
		page.NewCSS([]page.CSSContent{}),
		[]any{
			page.NewHTML("h2", []page.KeyValue{}, "Linuxus Login"),
			page.NewHTML("p", []page.KeyValue{{Key: "class", Value: "error"}}, "{{.Error}}").AddPrefix("{{if .Error}}").AddSuffix("{{end}}"),
			page.NewHTML("form", []page.KeyValue{{Key: "method", Value: "post"}, {Key: "action", Value: "/" + a.loginPath}},
				[]any{
					page.NewHTML("input", []page.KeyValue{{Key: "type", Value: "text"}, {Key: "name", Value: "id"}, {Key: "placeholder", Value: "ID"}, {Key: "required", Value: "true"}}, nil),
					page.NewHTML("input", []page.KeyValue{{Key: "type", Value: "password"}, {Key: "name", Value: "password"}, {Key: "placeholder", Value: "Password"}, {Key: "required", Value: "true"}}, nil),
					page.NewHTML("button", []page.KeyValue{{Key: "type", Value: "submit"}}, "Login"),
				},
			),
		},
	)

	return htmlpage.Render()
}

func (a *App) GetServicePage() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>linuxus shell</title>
    ` + a.GetServiceCSS() + `
</head>
<body>
    <div class="topbar">
        <div class="left">linuxus | {{.ID}}</div>
        <div class="right">
            <a class="btn" href="/` + a.terminalPath + `/" target="shellframe">Open Shell</a>
            <a class="btn btn-danger" href="/` + a.logoutPath + `">Logout</a>
        </div>
    </div>

    <div class="frame-wrap">
        <iframe name="shellframe" src="/` + a.terminalPath + `/"></iframe>
    </div>
</body>
</html>
`
}
