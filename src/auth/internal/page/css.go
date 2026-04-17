package page

// CSS represents a style block made of multiple CSS content entries.
type CSS struct {
	// contents stores individual CSS selector blocks.
	contents []*CSSContent
}

// NewCSS creates a CSS container with optional initial contents.
func NewCSS(contents ...*CSSContent) *CSS {
	return &CSS{
		contents: contents,
	}
}

// AddContents appends one or more CSS content blocks.
func (c *CSS) AddContents(content ...*CSSContent) *CSS {
	c.contents = append(c.contents, content...)
	return c
}

// RemoveContent removes the first content block matching the predicate.
func (c *CSS) RemoveContent(predicate func(x *CSSContent) bool) *CSS {
	for i, content := range c.contents {
		if predicate(content) {
			c.contents = append(c.contents[:i], c.contents[i+1:]...)
			break
		}
	}
	return c
}

// Contents returns all CSS content blocks.
func (c *CSS) Contents() []*CSSContent {
	return c.contents
}

// Render returns the complete style tag string.
func (c *CSS) Render() string {
	return c.renderWithIndent(0)
}

// renderWithIndent renders the style tag with indentation depth.
func (c *CSS) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := indentStr + "<style>\n"

	for _, content := range c.contents {
		cssStr += content.renderWithIndent(indent + 1)
	}

	cssStr += indentStr + "</style>\n"
	return cssStr
}
