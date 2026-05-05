package html

// CSS represents a collection of CSS selector blocks.
type CSS struct {
	// contents stores selector blocks rendered inside style tags.
	contents []*CSSContent
}

// NewCSS creates a CSS collection.
func NewCSS(contents ...*CSSContent) *CSS {
	return &CSS{
		contents: contents,
	}
}

// AddContents appends selector blocks to the CSS collection.
func (c *CSS) AddContents(content ...*CSSContent) *CSS {
	c.contents = append(c.contents, content...)
	return c
}

// RemoveContent removes the first selector block matching predicate.
func (c *CSS) RemoveContent(predicate func(x *CSSContent) bool) *CSS {
	for i, content := range c.contents {
		if predicate(content) {
			c.contents = append(c.contents[:i], c.contents[i+1:]...)
			break
		}
	}
	return c
}

// Contents returns selector blocks in this CSS collection.
func (c *CSS) Contents() []*CSSContent {
	return c.contents
}

// Render renders the CSS collection inside a style tag.
func (c *CSS) Render() string {
	return c.renderWithIndent(0)
}

// renderWithIndent renders CSS with the given indentation level.
func (c *CSS) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := indentStr + "<style>\n"

	for _, content := range c.contents {
		cssStr += content.renderWithIndent(indent + 1)
	}

	cssStr += indentStr + "</style>\n"
	return cssStr
}
