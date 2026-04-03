package handler

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
` + indentStr + `background: #545454;
` + indentStr + ``
	} else if !isDark {
		return `
` + indentStr + `background: #c0c0c0;
` + indentStr + ``
	}

	return ""
}
