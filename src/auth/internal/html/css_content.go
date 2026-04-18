package html

// CSSContent represents one CSS selector block with its attributes.
type CSSContent struct {
	// tag is the selector name.
	tag string
	// attributes are CSS properties inside the selector block.
	attributes []Attribute
}

// NewCSSContent creates a CSS selector block.
func NewCSSContent(name string, attributes ...Attribute) *CSSContent {
	return &CSSContent{
		tag:        name,
		attributes: attributes,
	}
}

// AddAttribute appends a CSS property to the selector block.
func (c *CSSContent) AddAttribute(key string, value string) *CSSContent {
	if c.attributes == nil {
		c.attributes = make([]Attribute, 0)
	}

	c.attributes = append(c.attributes, Attribute{Key: key, Value: value})
	return c
}

// RemoveAttribute removes the first attribute matching predicate.
func (c *CSSContent) RemoveAttribute(predicate func(x Attribute) bool) *CSSContent {
	for i, attr := range c.attributes {
		if predicate(attr) {
			c.attributes = append(c.attributes[:i], c.attributes[i+1:]...)
			break
		}
	}
	return c
}

// Attributes returns all attributes for this selector block.
func (c *CSSContent) Attributes() []Attribute {
	return c.attributes
}

// SetTag changes the selector tag.
func (c *CSSContent) SetTag(tag string) *CSSContent {
	c.tag = tag
	return c
}

// Tag returns the selector tag.
func (c *CSSContent) Tag() string {
	return c.tag
}

// Render renders this selector block to CSS text.
func (c *CSSContent) Render() string {
	return c.renderWithIndent(0)
}

// renderWithIndent renders this selector block with the given indentation level.
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
