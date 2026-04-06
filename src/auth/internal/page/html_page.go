package page

type HTMLPage struct {
	title string
	meta  []Attribute
	css   *CSS
	body  []any
}

func NewHTMLPage(title string, meta []Attribute, css *CSS, body ...any) *HTMLPage {
	return &HTMLPage{
		title: title,
		meta:  meta,
		css:   css,
		body:  body,
	}
}

func (p *HTMLPage) SetTitle(title string) *HTMLPage {
	p.title = title
	return p
}

func (p *HTMLPage) Title() string {
	return p.title
}

func (p *HTMLPage) AddMeta(key string, value string) *HTMLPage {
	if p.meta == nil {
		p.meta = make([]Attribute, 0)
	}

	p.meta = append(p.meta, Attribute{Key: key, Value: value})
	return p
}

func (p *HTMLPage) RemoveMeta(predicate func(x Attribute) bool) *HTMLPage {
	for i, kv := range p.meta {
		if predicate(kv) {
			p.meta = append(p.meta[:i], p.meta[i+1:]...)
			break
		}
	}
	return p
}

func (p *HTMLPage) Meta() []Attribute {
	return p.meta
}

func (p *HTMLPage) SetCSS(css *CSS) *HTMLPage {
	p.css = css
	return p
}

func (p *HTMLPage) CSS() *CSS {
	return p.css
}

func (p *HTMLPage) AddBodyContent(contents any) *HTMLPage {
	if p.body == nil {
		p.body = make([]any, 0)
	}

	p.body = append(p.body, contents)

	return p
}

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

func (p *HTMLPage) BodyContents() any {
	return p.body
}

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
