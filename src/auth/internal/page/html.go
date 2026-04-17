package page

// HTML represents a generic HTML element tree node.
type HTML struct {
	// tag is the HTML element name.
	tag string
	// attributes stores element attributes.
	attributes []Attribute
	// contents stores child content nodes and text.
	contents []any
	// prefixes stores strings rendered before the opening tag.
	prefixes []string
	// suffixes stores strings rendered after the closing tag.
	suffixes []string
}

// NewHTML creates an HTML element with attributes and optional contents.
func NewHTML(tag string, attributes []Attribute, contents ...any) *HTML {
	return &HTML{
		tag:        tag,
		attributes: attributes,
		contents:   contents,
		prefixes:   make([]string, 0),
		suffixes:   make([]string, 0),
	}
}

// SetTag updates the element tag name.
func (h *HTML) SetTag(tag string) *HTML {
	h.tag = tag
	return h
}

// Tag returns the element tag name.
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

// RemoveAttribute removes the first attribute matching the predicate.
func (h *HTML) RemoveAttribute(predicate func(x Attribute) bool) *HTML {
	for i, attr := range h.attributes {
		if predicate(attr) {
			h.attributes = append(h.attributes[:i], h.attributes[i+1:]...)
			break
		}
	}
	return h
}

// Attributes returns the element attributes.
func (h *HTML) Attributes() []Attribute {
	return h.attributes
}

// AddContent appends child content to the element.
func (h *HTML) AddContent(content any) *HTML {
	if h.contents == nil {
		h.contents = make([]any, 0)
		h.contents = append(h.contents, content)
		return h
	}

	h.contents = append(h.contents, content)
	return h
}

// RemoveContent removes the first child content that matches the predicate.
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

// AddPrefix appends a string rendered before the opening tag.
func (h *HTML) AddPrefix(prefix string) *HTML {
	if h.prefixes == nil {
		h.prefixes = make([]string, 0)
	}

	h.prefixes = append(h.prefixes, prefix)
	return h
}

// RemovePrefix removes the first prefix matching the predicate.
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

// AddSuffix appends a string rendered after the closing tag.
func (h *HTML) AddSuffix(suffix string) *HTML {
	if h.suffixes == nil {
		h.suffixes = make([]string, 0)
	}

	h.suffixes = append(h.suffixes, suffix)
	return h
}

// RemoveSuffix removes the first suffix matching the predicate.
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

// Contents returns the element child contents.
func (h *HTML) Contents() any {
	return h.contents
}

// Render returns the rendered HTML string.
func (h *HTML) Render() string {
	return h.renderWithIndent(0)
}

// renderWithIndent renders the element with indentation depth.
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

// getIndentStr returns tab-based indentation for the given depth.
func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "\t"
	}

	return indentStr
}
