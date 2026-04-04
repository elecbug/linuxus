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
` + indentStr + `display: ` + BOX_DISPLAY + `;
` + indentStr + `padding: ` + BOX_PADDING + `;
` + indentStr + `text-decoration: ` + BOX_TEXT_DECORATION + `;
` + indentStr + `border: ` + BOX_DARK_BORDER + `;
` + indentStr + `border-radius: ` + BOX_BORDER_RADIUS + `;
` + indentStr + `font-size: ` + BOX_FONT_SIZE + `;
` + indentStr + `color: ` + BOX_DARK_COLOR + `;
` + indentStr + `background: ` + BOX_DARK_BACKGROUND + `;
` + indentStr + ``
	} else if !isDark && !isDangerOpt {
		return `
` + indentStr + `display: ` + BOX_DISPLAY + `;
` + indentStr + `padding: ` + BOX_PADDING + `;
` + indentStr + `text-decoration: ` + BOX_TEXT_DECORATION + `;
` + indentStr + `border: ` + BOX_LIGHT_BORDER + `;
` + indentStr + `border-radius: ` + BOX_BORDER_RADIUS + `;
` + indentStr + `font-size: ` + BOX_FONT_SIZE + `;
` + indentStr + `color: ` + BOX_LIGHT_COLOR + `;
` + indentStr + `background: ` + BOX_LIGHT_BACKGROUND + `;
` + indentStr + ``
	} else if isDark && isDangerOpt {
		return `
` + indentStr + `display: ` + BOX_DISPLAY + `;
` + indentStr + `padding: ` + BOX_PADDING + `;
` + indentStr + `text-decoration: ` + BOX_TEXT_DECORATION + `;
` + indentStr + `border: ` + BOX_DANGER_BORDER + `;
` + indentStr + `border-radius: ` + BOX_BORDER_RADIUS + `;
` + indentStr + `font-size: ` + BOX_FONT_SIZE + `;
` + indentStr + `color: ` + BOX_DANGER_COLOR + `;
` + indentStr + `background: ` + BOX_DARK_BACKGROUND + `;
` + indentStr + ``
	} else if !isDark && isDangerOpt {
		return `
` + indentStr + `display: ` + BOX_DISPLAY + `;
` + indentStr + `padding: ` + BOX_PADDING + `;
` + indentStr + `text-decoration: ` + BOX_TEXT_DECORATION + `;
` + indentStr + `border: ` + BOX_DANGER_BORDER + `;
` + indentStr + `border-radius: ` + BOX_BORDER_RADIUS + `;
` + indentStr + `font-size: ` + BOX_FONT_SIZE + `;
` + indentStr + `color: ` + BOX_DANGER_COLOR + `;
` + indentStr + `background: ` + BOX_LIGHT_BACKGROUND + `;
` + indentStr + ``
	}

	return ""
}

func hoverEffect(indent int, isDark bool) string {
	indentStr := getIndentStr(indent)

	if isDark {
		return `
` + indentStr + `background: ` + BOX_HOVER_DARK_BACKGROUND + `;
` + indentStr + ``
	} else if !isDark {
		return `
` + indentStr + `background: ` + BOX_HOVER_LIGHT_BACKGROUND + `;
` + indentStr + ``
	}

	return ""
}
