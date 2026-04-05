package page

type HTMLPage struct {
	title    string
	meta     []KeyValue
	css      *CSS
	contents any
}

func NewHTMLPage(title string, meta []KeyValue, css *CSS, contents any) *HTMLPage {
	return &HTMLPage{
		title:    title,
		meta:     meta,
		css:      css,
		contents: contents,
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
		p.meta = make([]KeyValue, 0)
	}

	p.meta = append(p.meta, KeyValue{Key: key, Value: value})
	return p
}

func (p *HTMLPage) RemoveMeta(predicate func(x KeyValue) bool) *HTMLPage {
	for i, kv := range p.meta {
		if predicate(kv) {
			p.meta = append(p.meta[:i], p.meta[i+1:]...)
			break
		}
	}
	return p
}

func (p *HTMLPage) Meta() []KeyValue {
	return p.meta
}

func (p *HTMLPage) SetCSS(css *CSS) *HTMLPage {
	p.css = css
	return p
}

func (p *HTMLPage) CSS() *CSS {
	return p.css
}

func (p *HTMLPage) AddContent(contents any) *HTMLPage {
	if p.contents == nil {
		p.contents = contents
		return p
	}

	switch existingContent := p.contents.(type) {
	case string:
		p.contents = []any{existingContent, contents}
	case *HTML:
		p.contents = []any{existingContent, contents}
	case []any:
		p.contents = append(existingContent, contents)
	}
	return p
}

func (p *HTMLPage) RemoveContent(predicate func(x any) bool) *HTMLPage {
	if p.contents == nil {
		return p
	}

	switch existingContent := p.contents.(type) {
	case string:
		if predicate(existingContent) {
			p.contents = nil
		}
	case *HTML:
		if predicate(existingContent) {
			p.contents = nil
		}
	case []any:
		for i, content := range existingContent {
			if predicate(content) {
				p.contents = append(existingContent[:i], existingContent[i+1:]...)
				break
			}
		}
	}
	return p
}

func (p *HTMLPage) Contents() any {
	return p.contents
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
	if p.contents != nil {
		switch content := p.contents.(type) {
		case *HTML:
			pageStr += content.renderWithIndent(1)
		case string:
			pageStr += getIndentStr(1) + content + "\n"
		case []any:
			for _, c := range content {
				switch c := c.(type) {
				case *HTML:
					pageStr += c.renderWithIndent(1)
				case string:
					pageStr += getIndentStr(1) + c + "\n"
				}
			}
		}
	}
	pageStr += "</body>\n"
	pageStr += "</html>\n"

	return pageStr
}
