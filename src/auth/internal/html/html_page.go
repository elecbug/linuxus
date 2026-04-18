package html

// HTMLPage represents a full HTML document with head and body data.
type HTMLPage struct {
	// title is the page title rendered in head.
	title string
	// meta contains meta tag attributes.
	meta []Attribute
	// css contains optional embedded style definitions.
	css *CSS
	// body contains body child nodes or raw strings.
	body []any
}

// NewHTMLPage creates a full HTML document model.
func NewHTMLPage(title string, meta []Attribute, css *CSS, body ...any) *HTMLPage {
	return &HTMLPage{
		title: title,
		meta:  meta,
		css:   css,
		body:  body,
	}
}

// SetTitle updates the page title.
func (p *HTMLPage) SetTitle(title string) *HTMLPage {
	p.title = title
	return p
}

// Title returns the page title.
func (p *HTMLPage) Title() string {
	return p.title
}

// AddMeta appends a meta tag attribute pair.
func (p *HTMLPage) AddMeta(key string, value string) *HTMLPage {
	if p.meta == nil {
		p.meta = make([]Attribute, 0)
	}

	p.meta = append(p.meta, Attribute{Key: key, Value: value})
	return p
}

// RemoveMeta removes the first meta entry matching predicate.
func (p *HTMLPage) RemoveMeta(predicate func(x Attribute) bool) *HTMLPage {
	for i, kv := range p.meta {
		if predicate(kv) {
			p.meta = append(p.meta[:i], p.meta[i+1:]...)
			break
		}
	}
	return p
}

// Meta returns all meta tag attributes.
func (p *HTMLPage) Meta() []Attribute {
	return p.meta
}

// SetCSS sets the CSS definition for the page.
func (p *HTMLPage) SetCSS(css *CSS) *HTMLPage {
	p.css = css
	return p
}

// CSS returns the CSS definition for the page.
func (p *HTMLPage) CSS() *CSS {
	return p.css
}

// AddBodyContent appends content to the document body.
func (p *HTMLPage) AddBodyContent(contents any) *HTMLPage {
	if p.body == nil {
		p.body = make([]any, 0)
	}

	p.body = append(p.body, contents)

	return p
}

// RemoveBodyContent removes the first body item matching predicate.
func (p *HTMLPage) RemoveBodyContent(predicate func(x any) bool) *HTMLPage {
	if p.body == nil {
		return p
	}

	for i, content := range p.body {
		if predicate(content) {
			p.body = append(p.body[:i], p.body[i+1:]...)
			break
		}
	}

	return p
}

// BodyContents returns all body content values.
func (p *HTMLPage) BodyContents() any {
	return p.body
}

// Render renders the full HTML page string.
func (p *HTMLPage) Render() string {
	pageStr := "<!DOCTYPE html>\n"
	pageStr += "<html>\n"
	pageStr += "<head>\n"
	pageStr += getIndentStr(1) + "<title>" + p.title + "</title>\n"
	for _, kv := range p.meta {
		pageStr += getIndentStr(1) + "<meta " + kv.Key + "=\"" + kv.Value + "\">\n"
	}
	if p.css != nil {
		pageStr += p.css.renderWithIndent(1)
	}
	pageStr += "</head>\n"
	pageStr += "<body>\n"

	for _, content := range p.body {
		switch content := content.(type) {
		case nil:
			// No content to render
			break
		case string:
			if len(p.body) == 1 {
				pageStr += content
			} else {
				pageStr += getIndentStr(1) + content + "\n"
			}
		case *HTML:
			pageStr += content.renderWithIndent(1)
		}
	}

	pageStr += "</body>\n"
	pageStr += "</html>\n"

	return pageStr
}
