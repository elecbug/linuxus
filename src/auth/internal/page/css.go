package page

type CSS struct {
	contents []CSSContent
}

func NewCSS(contents []CSSContent) *CSS {
	return &CSS{
		contents: contents,
	}
}

func (c *CSS) AddContent(content CSSContent) *CSS {
	c.contents = append(c.contents, content)
	return c
}

func (c *CSS) RemoveContent(predicate func(x CSSContent) bool) *CSS {
	for i, content := range c.contents {
		if predicate(content) {
			c.contents = append(c.contents[:i], c.contents[i+1:]...)
			break
		}
	}
	return c
}

func (c *CSS) Contents() []CSSContent {
	return c.contents
}

func (c *CSS) Render() string {
	return c.renderWithIndent(0)
}

func (c *CSS) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := indentStr + "<style>\n"

	for _, content := range c.contents {
		cssStr += content.renderWithIndent(indent + 1)
	}

	cssStr += indentStr + "</style>\n"
	return cssStr
}
