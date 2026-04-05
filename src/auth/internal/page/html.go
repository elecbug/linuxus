package page

type HTML struct {
	tag        string
	attributes []KeyValue
	contents   any
	prefixes   []string
	suffixes   []string
}

func NewHTML(tag string, attributes []KeyValue, contents any) *HTML {
	return &HTML{
		tag:        tag,
		attributes: attributes,
		contents:   contents,
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
		h.attributes = make([]KeyValue, 0)
	}

	h.attributes = append(h.attributes, KeyValue{Key: key, Value: value})
	return h
}

func (h *HTML) RemoveAttribute(predicate func(x KeyValue) bool) *HTML {
	for i, attr := range h.attributes {
		if predicate(attr) {
			h.attributes = append(h.attributes[:i], h.attributes[i+1:]...)
			break
		}
	}
	return h
}

func (h *HTML) Attributes() []KeyValue {
	return h.attributes
}

func (h *HTML) AddContent(contents any) *HTML {
	if h.contents == nil {
		h.contents = contents
		return h
	}

	switch existingContent := h.contents.(type) {
	case string:
		h.contents = []any{existingContent, contents}
	case *HTML:
		h.contents = []any{existingContent, contents}
	case []any:
		h.contents = append(existingContent, contents)
	}
	return h
}

func (h *HTML) RemoveContent(predicate func(x any) bool) *HTML {
	if h.contents == nil {
		return h
	}

	switch existingContent := h.contents.(type) {
	case string:
		if predicate(existingContent) {
			h.contents = nil
		}
	case *HTML:
		if predicate(existingContent) {
			h.contents = nil
		}
	case []any:
		for i, content := range existingContent {
			if predicate(content) {
				h.contents = append(existingContent[:i], existingContent[i+1:]...)
				break
			}
		}
	}
	return h
}

func (h *HTML) AddPrefix(prefix string) *HTML {
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
		for _, prefix := range h.prefixes {
			htmlStr += indentStr + prefix + "\n"
		}
	}

	htmlStr += indentStr + "<" + h.tag
	for _, attr := range h.attributes {
		htmlStr += " " + attr.Key + `="` + attr.Value + `"`
	}
	htmlStr += ">\n"

	switch content := h.contents.(type) {
	case nil:
		// No content to render
		break
	case string:
		htmlStr += indentStr + getIndentStr(1) + content + "\n"
	case *HTML:
		htmlStr += content.renderWithIndent(indent + 1)
	case []any:
		for _, item := range content {
			if htmlItem, ok := item.(*HTML); ok {
				htmlStr += htmlItem.renderWithIndent(indent + 1)
			} else if strItem, ok := item.(string); ok {
				htmlStr += indentStr + getIndentStr(1) + strItem + "\n"
			}
		}
	}

	htmlStr += indentStr + "</" + h.tag + ">\n"

	if h.suffixes != nil {
		for _, suffix := range h.suffixes {
			htmlStr += indentStr + suffix + "\n"
		}
	}

	return htmlStr
}

func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "\t"
	}

	return indentStr
}
