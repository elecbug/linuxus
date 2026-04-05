package page

type HTML struct {
	tag        string
	attributes map[string]string
	content    any
}

func newHTML(tag string, attributes map[string]string, content any) *HTML {
	return &HTML{
		tag:        tag,
		attributes: attributes,
		content:    content,
	}
}

func (h *HTML) Render() string {
	return h.renderWithIndent(0)
}

func (h *HTML) renderWithIndent(indent int) string {
	indentStr := getIndentStr(indent)
	htmlStr := ""

	// Render opening tag with attributes
	htmlStr += indentStr + "<" + h.tag
	for key, value := range h.attributes {
		htmlStr += " " + key + `="` + value + `"`
	}
	htmlStr += ">\n"

	// Render content
	switch content := h.content.(type) {
	case string:
		htmlStr += indentStr + "    " + content + "\n"
	case *HTML:
		htmlStr += content.renderWithIndent(indent + 1)
	case []any:
		for _, item := range content {
			if htmlItem, ok := item.(*HTML); ok {
				htmlStr += htmlItem.renderWithIndent(indent + 1)
			} else if strItem, ok := item.(string); ok {
				htmlStr += indentStr + "    " + strItem + "\n"
			}
		}
	}

	// Render closing tag
	htmlStr += indentStr + "</" + h.tag + ">\n"

	return htmlStr
}

func getIndentStr(indent int) string {
	indentStr := ""

	for i := 0; i < indent; i++ {
		indentStr += "    "
	}

	return indentStr
}
