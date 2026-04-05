package page

type CSSContent struct {
	tag        string
	class      string
	attributes []KeyValue
}

func NewCSSContent(name string, class string, attributes ...KeyValue) *CSSContent {
	return &CSSContent{
		tag:        name,
		class:      class,
		attributes: attributes,
	}
}

func (c *CSSContent) AddAttribute(key string, value string) *CSSContent {
	if c.attributes == nil {
		c.attributes = make([]KeyValue, 0)
	}

	c.attributes = append(c.attributes, KeyValue{Key: key, Value: value})
	return c
}

func (c *CSSContent) RemoveAttribute(predicate func(x KeyValue) bool) *CSSContent {
	for i, attr := range c.attributes {
		if predicate(attr) {
			c.attributes = append(c.attributes[:i], c.attributes[i+1:]...)
			break
		}
	}
	return c
}

func (c *CSSContent) Attributes() []KeyValue {
	return c.attributes
}

func (c *CSSContent) SetTag(tag string) *CSSContent {
	c.tag = tag
	return c
}

func (c *CSSContent) Tag() string {
	return c.tag
}

func (c *CSSContent) SetClass(class string) *CSSContent {
	c.class = class
	return c
}

func (c *CSSContent) Class() string {
	return c.class
}

func (c *CSSContent) Render() string {
	return c.renderWithIndent(0)
}

func (c *CSSContent) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	cssStr := ""

	cssStr += indentStr + c.tag + "." + c.class + " {\n"
	for _, attr := range c.attributes {
		cssStr += indentStr + getIndentStr(1) + attr.Key + ": " + attr.Value + ";\n"
	}
	cssStr += indentStr + "}\n"

	return cssStr
}
