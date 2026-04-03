package handler

func (a *App) GetLoginPage() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Linuxus Login</title>
    ` + a.GetLoginCSS() + `
</head>
<body>
    <h2>Linuxus Login</h2>
    {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
    <form method="post" action="/` + a.loginPath + `">
        <input type="text" name="id" placeholder="ID" required>
        <input type="password" name="password" placeholder="Password" required>
        <button type="submit">Login</button>
    </form>
</body>
</html>
`
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

/*
   .links {
       margin-top: 16px;
       display: flex;
       gap: 12px;
       color: white;
   }
   .links a {
       color: white;
       text-decoration: none;
   }
   .links a:hover {
       text-decoration: underline;
   }
   .links a:active {
       color: #ccccff;
   }
   .links a:visited {
       color: #9999ff;
   }
*/
