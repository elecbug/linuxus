package handler

func (a *App) GetLoginCSS() string {
	return `
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
`
}

func (a *App) GetServiceCSS() string {
	return `
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
`
}

func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "    "
	}

	return indentStr
}

func baseBox(indent int, isDark, isDangerOpt bool) string {
	indentStr := getIndentStr(indent)

	if isDark && !isDangerOpt {
		return `
` + indentStr + `display: inline-block;
` + indentStr + `padding: 8px 12px;
` + indentStr + `text-decoration: none;
` + indentStr + `border: 1px solid #999999;
` + indentStr + `border-radius: 6px;
` + indentStr + `font-size: 16px;
` + indentStr + `color: white;
` + indentStr + `background: #242424;
` + indentStr + ``
	} else if !isDark && !isDangerOpt {
		return `
` + indentStr + `display: inline-block;
` + indentStr + `padding: 8px 12px;
` + indentStr + `text-decoration: none;
` + indentStr + `border: 1px solid #555555;
` + indentStr + `border-radius: 6px;
` + indentStr + `font-size: 16px;
` + indentStr + `color: black;
` + indentStr + `background: #f0f0f0;
` + indentStr + ``
	} else if isDark && isDangerOpt {
		return `
` + indentStr + `display: inline-block;
` + indentStr + `padding: 8px 12px;
` + indentStr + `text-decoration: none;
` + indentStr + `border: 1px solid #ff0000;
` + indentStr + `border-radius: 6px;
` + indentStr + `font-size: 16px;
` + indentStr + `color: #ff0000;
` + indentStr + `background: #242424;
` + indentStr + ``
	} else if !isDark && isDangerOpt {
		return `
` + indentStr + `display: inline-block;
` + indentStr + `padding: 8px 12px;
` + indentStr + `text-decoration: none;
` + indentStr + `border: 1px solid #ff0000;
` + indentStr + `border-radius: 6px;
` + indentStr + `font-size: 16px;
` + indentStr + `color: #ff0000;
` + indentStr + `background: #f0f0f0;
` + indentStr + ``
	}

	return ""
}

func hoverEffect(indent int, isDark bool) string {
	indentStr := getIndentStr(indent)

	if isDark {
		return `
` + indentStr + `background: #343434;
` + indentStr + ``
	} else if !isDark {
		return `
` + indentStr + `background: #e0e0e0;
` + indentStr + ``
	}

	return ""
}
