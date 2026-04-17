package page

// CSSContent represents one CSS selector block and its declarations.
type CSSContent struct {
	// tag is the selector for the CSS block.
	tag string
	// attributes stores CSS property/value pairs for the selector.
	attributes []Attribute
}

// NewCSSContent creates a selector block with optional declarations.
func NewCSSContent(name string, attributes ...Attribute) *CSSContent {
	return &CSSContent{
		tag:        name,
		attributes: attributes,
	}
}

// AddAttribute appends a CSS declaration to the block.
func (c *CSSContent) AddAttribute(key string, value string) *CSSContent {
	if c.attributes == nil {
		c.attributes = make([]Attribute, 0)
	}

	c.attributes = append(c.attributes, Attribute{Key: key, Value: value})
	return c
}

// RemoveAttribute removes the first declaration that matches the predicate.
func (c *CSSContent) RemoveAttribute(predicate func(x Attribute) bool) *CSSContent {
	for i, attr := range c.attributes {
		if predicate(attr) {
			c.attributes = append(c.attributes[:i], c.attributes[i+1:]...)
			break
		}
	}
	return c
}

// Attributes returns all declarations on the selector block.
func (c *CSSContent) Attributes() []Attribute {
	return c.attributes
}

// SetTag updates the selector name.
func (c *CSSContent) SetTag(tag string) *CSSContent {
	c.tag = tag
	return c
}

// Tag returns the selector name.
func (c *CSSContent) Tag() string {
	return c.tag
}

// Render returns the rendered selector block string.
func (c *CSSContent) Render() string {
	return c.renderWithIndent(0)
}

// renderWithIndent renders the selector block with indentation depth.
func (c *CSSContent) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := ""

	cssStr += indentStr + c.tag + " {\n"
	for _, attr := range c.attributes {
		cssStr += indentStr + getIndentStr(1) + attr.Key + ": " + attr.Value + ";\n"
	}
	cssStr += indentStr + "}\n"

	return cssStr
}
