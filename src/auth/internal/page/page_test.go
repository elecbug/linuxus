package page_test

import (
	"testing"

	"github.com/elecbug/linuxus/src/auth/internal/page"
)

func TestCSSRendering(t *testing.T) {
	css := page.NewCSS(
		page.NewCSSContent("body.main",
			page.NewAttributes(
				"background-color", "#f0f0f0",
				"color", "#242424",
				"font-family", "Arial, sans-serif",
			)...,
		),
		page.NewCSSContent("h2.title",
			page.NewAttributes(
				"color", "#333333",
			)...,
		),
		page.NewCSSContent("p.error",
			page.NewAttributes(
				"color", "red",
			)...,
		),
		page.NewCSSContent("form.login-form",
			page.NewAttributes(
				"display", "flex",
				"flex-direction", "column",
				"width", "300px",
				"margin", "0 auto",
			)...,
		),
		page.NewCSSContent("input.input-field",
			page.NewAttributes(
				"padding", "10px",
				"margin", "10px 0",
				"border", "1px solid #cccccc",
				"border-radius", "4px",
			)...,
		),
	)

	cssStr := css.Render()

	expected := `<style>
	body.main {
		background-color: #f0f0f0;
		color: #242424;
		font-family: Arial, sans-serif;
	}
	h2.title {
		color: #333333;
	}
	p.error {
		color: red;
	}
	form.login-form {
		display: flex;
		flex-direction: column;
		width: 300px;
		margin: 0 auto;
	}
	input.input-field {
		padding: 10px;
		margin: 10px 0;
		border: 1px solid #cccccc;
		border-radius: 4px;
	}
</style>
`
	if cssStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s",
			expected,
			cssStr,
		)
	}
}

func TestHTMLPageRendering(t *testing.T) {
	css := page.NewCSS(
		page.NewCSSContent("body.main",
			page.NewAttributes(
				"background-color", "#f0f0f0",
				"color", "#242424",
				"font-family", "Arial, sans-serif",
			)...,
		),
	)

	htmlContent := page.NewHTML(
		"div",
		page.NewAttributes("class", "container"),
		"Hello, World!",
	)

	p := page.NewHTMLPage(
		"Test Page",
		page.NewAttributes("charset", "UTF-8"),
		css,
		htmlContent,
	)

	p.AddBodyContent(
		page.NewHTML(
			"div",
			page.NewAttributes("class", "footer"),
			"Footer content",
		),
	)

	htmlContent.AddContent(page.NewHTML(
		"div",
		page.NewAttributes("class", "section"),
		"Section content",
	))

	pageStr := p.Render()

	expected := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta charset="UTF-8">
	<style>
		body.main {
			background-color: #f0f0f0;
			color: #242424;
			font-family: Arial, sans-serif;
		}
	</style>
</head>
<body>
	<div class="container">
		Hello, World!
		<div class="section">Section content</div>
	</div>
	<div class="footer">Footer content</div>
</body>
</html>
`
	if pageStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s",
			expected,
			pageStr,
		)
	}
}

func TestHTMLPageRemoveContent(t *testing.T) {
	htmlContent := page.NewHTML(
		"div",
		page.NewAttributes("class", "container"),
		"Hello, World!",
	)
	p := page.NewHTMLPage("Test Page", nil, nil, htmlContent)

	p.AddBodyContent(
		page.NewHTML(
			"div",
			page.NewAttributes("class", "footer"),
			"Footer content",
		),
	)
	p.RemoveBodyContent(func(x any) bool {
		if html, ok := x.(*page.HTML); ok {
			for _, attr := range html.Attributes() {
				if attr.Key == "class" && attr.Value == "footer" {
					return true
				}
			}
		}
		return false
	})

	pageStr := p.Render()

	expected := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<div class="container">Hello, World!</div>
</body>
</html>
`
	if pageStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s",
			expected,
			pageStr,
		)
	}
}
