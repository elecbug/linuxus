package page

type CSS struct {
	contents []CSSContent
}

func newCSS(contents []CSSContent) *CSS {
	return &CSS{
		contents: contents,
	}
}

func (c *CSS) Render() string {
	return c.renderWithIndent(0)
}

func (c *CSS) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := indentStr + "<style>\n"

	for _, content := range c.contents {
		cssStr += content.renderWithIndent(indent+1) + "\n"
	}

	cssStr += indentStr + "</style>\n"
	return cssStr
}
