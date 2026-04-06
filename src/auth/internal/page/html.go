package page

type HTML struct {
	tag        string
	attributes []Attribute
	contents   []any
	prefixes   []string
	suffixes   []string
}

func NewHTML(tag string, attributes []Attribute, contents ...any) *HTML {
	return &HTML{
		tag:        tag,
		attributes: attributes,
		contents:   contents,
		prefixes:   make([]string, 0),
		suffixes:   make([]string, 0),
	}
}

func (h *HTML) SetTag(tag string) *HTML {
	h.tag = tag
	return h
}

func (h *HTML) Tag() string {
	return h.tag
}

func (h *HTML) AddAttribute(key string, value string) *HTML {
	if h.attributes == nil {
		h.attributes = make([]Attribute, 0)
	}

	h.attributes = append(h.attributes, Attribute{Key: key, Value: value})
	return h
}

func (h *HTML) RemoveAttribute(predicate func(x Attribute) bool) *HTML {
	for i, attr := range h.attributes {
		if predicate(attr) {
			h.attributes = append(h.attributes[:i], h.attributes[i+1:]...)
			break
		}
	}
	return h
}

func (h *HTML) Attributes() []Attribute {
	return h.attributes
}

func (h *HTML) AddContent(content any) *HTML {
	if h.contents == nil {
		h.contents = make([]any, 0)
		h.contents = append(h.contents, content)
		return h
	}

	h.contents = append(h.contents, content)
	return h
}

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

func (h *HTML) AddPrefix(prefix string) *HTML {
	if h.prefixes == nil {
		h.prefixes = make([]string, 0)
	}

	h.prefixes = append(h.prefixes, prefix)
	return h
}

func (h *HTML) RemovePrefix(predicate func(x string) bool) *HTML {
	for i, prefix := range h.prefixes {
		if predicate(prefix) {
			h.prefixes = append(h.prefixes[:i], h.prefixes[i+1:]...)
			break
		}
	}
	return h
}

func (h *HTML) Prefixes() []string {
	return h.prefixes
}

func (h *HTML) AddSuffix(suffix string) *HTML {
	if h.suffixes == nil {
		h.suffixes = make([]string, 0)
	}

	h.suffixes = append(h.suffixes, suffix)
	return h
}

func (h *HTML) RemoveSuffix(predicate func(x string) bool) *HTML {
	for i, suffix := range h.suffixes {
		if predicate(suffix) {
			h.suffixes = append(h.suffixes[:i], h.suffixes[i+1:]...)
			break
		}
	}
	return h
}

func (h *HTML) Suffixes() []string {
	return h.suffixes
}

func (h *HTML) Contents() any {
	return h.contents
}

func (h *HTML) Render() string {
	return h.renderWithIndent(0)
}

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

func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "\t"
	}

	return indentStr
}
