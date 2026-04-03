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
        }
        form {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        input {
            padding: 10px;
            font-size: 16px;
        }
        button {
            padding: 10px;
            font-size: 16px;
        }
        .error {
            color: red;
        }
        .links {
            margin-top: 16px;
            display: flex;
            gap: 12px;
        }
    </style>
</head>
<body>
    <h2>Linuxus Login</h2>
    {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
    <form method="post" action="/` + a.loginPath + `">
        <input type="text" name="student_id" placeholder="Student ID" required>
        <input type="password" name="password" placeholder="Password" required>
        <button type="submit">Login</button>
    </form>
    <div class="links">
        <a href="/` + a.servicePath + `/">Go to service</a>
        <a href="/` + a.logoutPath + `">Logout</a>
    </div>
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
            border-bottom: 1px solid #ddd;
            background: #595d65;
			color: white;
        }

        .left {
            font-weight: bold;
        }

        .right {
            display: flex;
            gap: 10px;
        }

        .btn {
            display: inline-block;
            padding: 8px 12px;
            text-decoration: none;
            border: 1px solid #999;
            border-radius: 6px;
            color: white;
            background: #242529;
        }

        .btn-danger {
            border-color: #c33;
            color: #c33;
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
    </style>
</head>
<body>
    <div class="topbar">
        <div class="left">linuxus | {{.StudentID}}</div>
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
