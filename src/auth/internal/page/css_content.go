package page

type CSSContent struct {
	tag        string
	class      string
	attributes map[string]string
}

func newCSSContent(name string, class string, contents map[string]string) *CSSContent {
	return &CSSContent{
		tag:        name,
		class:      class,
		attributes: contents,
	}
}

func (c *CSSContent) Render() string {
	return c.renderWithIndent(0)
}

func (c *CSSContent) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := ""

	cssStr += indentStr + c.tag + "." + c.class + " {\n"
	for key, value := range c.attributes {
		cssStr += indentStr + "    " + key + ": " + value + ";\n"
	}
	cssStr += indentStr + "}\n"

	return cssStr
}
