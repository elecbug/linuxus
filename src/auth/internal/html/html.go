package html

// HTML represents an HTML element with attributes and child contents.
type HTML struct {
	// tag is the element tag name.
	tag string
	// attributes contains element attributes.
	attributes []Attribute
	// contents contains child nodes or text values.
	contents []any
	// prefixes are raw strings rendered before the opening tag.
	prefixes []string
	// suffixes are raw strings rendered after the closing tag.
	suffixes []string
}

// NewHTML creates a new HTML element.
func NewHTML(tag string, attributes []Attribute, contents ...any) *HTML {
	return &HTML{
		tag:        tag,
		attributes: attributes,
		contents:   contents,
		prefixes:   make([]string, 0),
		suffixes:   make([]string, 0),
	}
}

// SetTag updates the element tag.
func (h *HTML) SetTag(tag string) *HTML {
	h.tag = tag
	return h
}

// Tag returns the element tag.
func (h *HTML) Tag() string {
	return h.tag
}

// AddAttribute appends an attribute to the element.
func (h *HTML) AddAttribute(key string, value string) *HTML {
	if h.attributes == nil {
		h.attributes = make([]Attribute, 0)
	}

	h.attributes = append(h.attributes, Attribute{Key: key, Value: value})
	return h
}

// RemoveAttribute removes the first attribute matching predicate.
func (h *HTML) RemoveAttribute(predicate func(x Attribute) bool) *HTML {
	for i, attr := range h.attributes {
		if predicate(attr) {
			h.attributes = append(h.attributes[:i], h.attributes[i+1:]...)
			break
		}
	}
	return h
}

// Attributes returns element attributes.
func (h *HTML) Attributes() []Attribute {
	return h.attributes
}

// AddContent appends a child content item.
func (h *HTML) AddContent(content any) *HTML {
	if h.contents == nil {
		h.contents = make([]any, 0)
		h.contents = append(h.contents, content)
		return h
	}

	h.contents = append(h.contents, content)
	return h
}

// RemoveContent removes the first child content matching predicate.
func (h *HTML) RemoveContent(predicate func(x any) bool) *HTML {
	if h.contents == nil {
		return h
	}

	for i, content := range h.contents {
		if predicate(content) {
			h.contents = append(h.contents[:i], h.contents[i+1:]...)
			break
		}
	}
	return h
}

// AddPrefix appends raw content rendered before the opening tag.
func (h *HTML) AddPrefix(prefix string) *HTML {
	if h.prefixes == nil {
		h.prefixes = make([]string, 0)
	}

	h.prefixes = append(h.prefixes, prefix)
	return h
}

// RemovePrefix removes the first prefix matching predicate.
func (h *HTML) RemovePrefix(predicate func(x string) bool) *HTML {
	for i, prefix := range h.prefixes {
		if predicate(prefix) {
			h.prefixes = append(h.prefixes[:i], h.prefixes[i+1:]...)
			break
		}
	}
	return h
}

// Prefixes returns all configured prefixes.
func (h *HTML) Prefixes() []string {
	return h.prefixes
}

// AddSuffix appends raw content rendered after the closing tag.
func (h *HTML) AddSuffix(suffix string) *HTML {
	if h.suffixes == nil {
		h.suffixes = make([]string, 0)
	}

	h.suffixes = append(h.suffixes, suffix)
	return h
}

// RemoveSuffix removes the first suffix matching predicate.
func (h *HTML) RemoveSuffix(predicate func(x string) bool) *HTML {
	for i, suffix := range h.suffixes {
		if predicate(suffix) {
			h.suffixes = append(h.suffixes[:i], h.suffixes[i+1:]...)
			break
		}
	}
	return h
}

// Suffixes returns all configured suffixes.
func (h *HTML) Suffixes() []string {
	return h.suffixes
}

// Contents returns child content values.
func (h *HTML) Contents() any {
	return h.contents
}

// Render renders the element as formatted HTML.
func (h *HTML) Render() string {
	return h.renderWithIndent(0)
}

// renderWithIndent renders the element at the given indentation level.
func (h *HTML) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	htmlStr := ""

	if h.prefixes != nil {
		htmlStr += indentStr
		for _, prefix := range h.prefixes {
			htmlStr += prefix
		}
	} else {
		htmlStr += indentStr
	}

	htmlStr += "<" + h.tag
	for _, attr := range h.attributes {
		htmlStr += " " + attr.Key + `="` + attr.Value + `"`
	}
	htmlStr += ">"

	if len(h.contents) >= 2 {
		htmlStr += "\n"
	}

	for _, content := range h.contents {
		switch content := content.(type) {
		case nil:
			// No content to render
			break
		case string:
			if len(h.contents) == 1 {
				htmlStr += content
			} else {
				htmlStr += getIndentStr(indent+1) + content + "\n"
			}
		case *HTML:
			htmlStr += content.renderWithIndent(indent + 1)
		}
	}

	if len(h.contents) >= 2 {
		htmlStr += indentStr + "</" + h.tag + ">"
	} else if len(h.contents) == 1 {
		if _, ok := h.contents[0].(string); ok {
			htmlStr += "</" + h.tag + ">"
		} else {
			htmlStr += "\n" + indentStr + "</" + h.tag + ">"
		}
	}

	if h.suffixes != nil {
		for _, suffix := range h.suffixes {
			htmlStr += suffix
		}
	}

	htmlStr += "\n"

	return htmlStr
}

// getIndentStr returns a tab-based indentation string for the level.
func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "\t"
	}

	return indentStr
}
