package handler

func (a *App) GetLoginPage() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Linuxus Login</title>
    <style>
        body {
            font-family: sans-serif;
            max-width: 420px;
            margin: 60px auto;
            background: #2e2e2e;
            color: white;
        }
        form {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        input {` + baseBox(3, false, false) + `}
        button {` + baseBox(3, true, false) + `}
        button:hover {` + hoverEffect(3, true) + `}
        .error {
            color: #ff0000;
        }
    </style>
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
    <style>
        html, body {
            margin: 0;
            padding: 0;
            height: 100%;
            font-family: sans-serif;
        }
        .topbar {
            height: 56px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 16px;
            box-sizing: border-box;
            border-bottom: 1px solid #dddddd;
            background: #595959;
			color: white;
        }
        .left {
            font-weight: bold;
        }
        .right {
            display: flex;
            gap: 10px;
        }
        .frame-wrap {
            height: calc(100% - 56px);
        }
        iframe {
            width: 100%;
            height: 100%;
            border: 0;
            display: block;
        }
        .btn {` + baseBox(3, true, false) + `}
        .btn:hover {` + hoverEffect(3, true) + `}
        .btn-danger {` + baseBox(3, true, true) + `}
        .btn-danger:hover {` + hoverEffect(3, true) + `}
    </style>
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
