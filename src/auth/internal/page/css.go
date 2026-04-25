package page

import (
	"fmt"

	html "github.com/elecbug/linuxus/src/auth/internal/html"
)

// getLoginCSS builds style rules for the login page.
func getLoginCSS() *html.CSS {
	return html.NewCSS().AddContents(
		loginBodyCSS(),
		loginErrorCSS(),
	).AddContents(
		loginTooltipCSS()...,
	).AddContents(
		footerCSS(40)...,
	).AddContents(
		loginFormCSS()...,
	).AddContents(
		loginButtonCSS()...,
	)
}

// getErrorCSS builds style rules for the error page.
func getErrorCSS() *html.CSS {
	return html.NewCSS().AddContents(
		loginBodyCSS(),
		loginErrorCSS(),
	).AddContents(
		loginTooltipCSS()...,
	).AddContents(
		footerCSS(40)...,
	)
}

// getServiceCSS builds style rules for the service page.
func getServiceCSS() *html.CSS {
	return html.NewCSS().AddContents(
		serviceBodyCSS(),
	).AddContents(
		serviceToolBarCSS()...,
	).AddContents(
		serviceIframeCSS()...,
	).AddContents(
		serviceButtonCSS()...,
	).AddContents(
		footerCSS(20)...,
	)

}

// footerCSS returns footer style rules with configurable top margin.
func footerCSS(marginTop int) []*html.CSSContent {
	return []*html.CSSContent{
		html.NewCSSContent("footer",
			html.NewAttributes(
				"margin-top", fmt.Sprintf("%dpx", marginTop),
				"margin-bottom", "20px",
				"font-size", "0.9em",
				"text-align", "center",
				"color", FOOTER_COLOR,
			)...,
		),
		html.NewCSSContent("footer a",
			html.NewAttributes(
				"color", FOOTER_LINK_COLOR,
				"text-decoration", "none",
			)...,
		),
		html.NewCSSContent("footer a:hover",
			html.NewAttributes(
				"color", FOOTER_LINK_HOVER_COLOR,
				"text-decoration", "underline",
			)...,
		),
		html.NewCSSContent("footer a:active",
			html.NewAttributes(
				"color", FOOTER_LINK_ACTIVE_COLOR,
				"text-decoration", "underline",
			)...,
		),
		html.NewCSSContent("footer a:visited",
			html.NewAttributes(
				"color", FOOTER_LINK_VISITED_COLOR,
				"text-decoration", "underline",
			)...,
		),
	}
}

// loginBodyCSS returns base body styling for login and error pages.
func loginBodyCSS() *html.CSSContent {
	return html.NewCSSContent("body",
		html.NewAttributes(
			"font-family", BASE_FONT_FAMILY,
			"max-width", "420px",
			"margin", "60px auto",
			"background", BASE_BACKGROUND,
			"color", BASE_COLOR,
		)...,
	)
}

// loginFormCSS returns styling for the login form container.
func loginFormCSS() []*html.CSSContent {
	return []*html.CSSContent{
		html.NewCSSContent("form.login-form",
			html.NewAttributes(
				"display", "flex",
				"flex-direction", "column",
				"gap", "12px",
			)...,
		),
	}
}

// loginErrorCSS returns styling for login error messages.
func loginErrorCSS() *html.CSSContent {
	return html.NewCSSContent("p.error",
		html.NewAttributes(
			"color", DANGER_COLOR,
		)...,
	)
}

// loginTooltipCSS returns styling for helper tooltip text.
func loginTooltipCSS() []*html.CSSContent {
	return []*html.CSSContent{
		html.NewCSSContent("p.tooltip",
			html.NewAttributes(
				"color", TOOLTIP_COLOR,
				"font-size", "0.9em",
				"font-style", "italic",
			)...,
		),
		html.NewCSSContent("a.signup-link",
			html.NewAttributes(
				"color", FOOTER_LINK_COLOR,
				"text-decoration", "none",
			)...,
		),
		html.NewCSSContent("a.signup-link:hover",
			html.NewAttributes(
				"color", FOOTER_LINK_HOVER_COLOR,
				"text-decoration", "underline",
			)...,
		),
		html.NewCSSContent("a.signup-link:active",
			html.NewAttributes(
				"color", FOOTER_LINK_ACTIVE_COLOR,
				"text-decoration", "underline",
			)...,
		),
		html.NewCSSContent("a.signup-link:visited",
			html.NewAttributes(
				"color", FOOTER_LINK_VISITED_COLOR,
				"text-decoration", "underline",
			)...,
		),
	}
}

// loginButtonCSS returns styling for login form inputs and button.
func loginButtonCSS() []*html.CSSContent {
	return []*html.CSSContent{
		box("input", false, false),
		box("button", true, false),
		boxHoverEffect("button:hover", true),
	}
}

// serviceBodyCSS returns base layout styling for service page.
func serviceBodyCSS() *html.CSSContent {
	return html.NewCSSContent("html, body",
		html.NewAttributes(
			"margin", "0",
			"padding", "0",
			"height", "100%",
			"display", "flex",
			"flex-direction", "column",
			"font-family", BASE_FONT_FAMILY,
			"background", BASE_BACKGROUND,
			"color", BASE_COLOR,
		)...,
	)
}

// serviceToolBarCSS returns style rules for service top toolbar.
func serviceToolBarCSS() []*html.CSSContent {
	return []*html.CSSContent{
		html.NewCSSContent("div.topbar",
			html.NewAttributes(
				"height", "56px",
				"flex-shrink", "0",
				"display", "flex",
				"align-items", "center",
				"justify-content", "space-between",
				"padding", "0 16px",
				"box-sizing", "border-box",
				"border-bottom", DARK_BORDER,
			)...,
		),
		html.NewCSSContent("div.left",
			html.NewAttributes(
				"font-size", "1.2em",
				"font-weight", "bold",
			)...,
		),
		html.NewCSSContent("div.right",
			html.NewAttributes(
				"display", "flex",
				"gap", "10px",
			)...,
		),
	}
}

// serviceIframeCSS returns style rules for terminal iframe area.
func serviceIframeCSS() []*html.CSSContent {
	return []*html.CSSContent{
		html.NewCSSContent("div.frame-wrap",
			html.NewAttributes(
				"flex", "1",
				"min-height", "0",
			)...,
		),
		html.NewCSSContent("iframe",
			html.NewAttributes(
				"width", "100%",
				"height", "100%",
				"border", "0",
				"display", "block",
			)...,
		),
	}
}

// serviceButtonCSS returns style rules for service action buttons.
func serviceButtonCSS() []*html.CSSContent {
	return []*html.CSSContent{
		box("a.btn", true, false),
		boxHoverEffect("a.btn:hover", true),
		box("a.btn-danger", true, true),
		boxHoverEffect("a.btn-danger:hover", true),
	}
}

// box creates a reusable button/input style variant.
func box(tag string, isDark, isDangerOpt bool) *html.CSSContent {
	if isDark && !isDangerOpt {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"display", BOX_DISPLAY,
				"padding", BOX_PADDING,
				"text-decoration", BOX_TEXT_DECORATION,
				"border", DARK_BORDER,
				"border-radius", BOX_BORDER_RADIUS,
				"font-size", BOX_FONT_SIZE,
				"color", DARK_COLOR,
				"background", DARK_BACKGROUND,
			)...,
		)
	} else if !isDark && !isDangerOpt {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"display", BOX_DISPLAY,
				"padding", BOX_PADDING,
				"text-decoration", BOX_TEXT_DECORATION,
				"border", LIGHT_BORDER,
				"border-radius", BOX_BORDER_RADIUS,
				"font-size", BOX_FONT_SIZE,
				"color", LIGHT_COLOR,
				"background", LIGHT_BACKGROUND,
			)...,
		)
	} else if isDark && isDangerOpt {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"display", BOX_DISPLAY,
				"padding", BOX_PADDING,
				"text-decoration", BOX_TEXT_DECORATION,
				"border", DANGER_BORDER,
				"border-radius", BOX_BORDER_RADIUS,
				"font-size", BOX_FONT_SIZE,
				"color", DANGER_COLOR,
				"background", DARK_BACKGROUND,
			)...,
		)
	} else if !isDark && isDangerOpt {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"display", BOX_DISPLAY,
				"padding", BOX_PADDING,
				"text-decoration", BOX_TEXT_DECORATION,
				"border", DANGER_BORDER,
				"border-radius", BOX_BORDER_RADIUS,
				"font-size", BOX_FONT_SIZE,
				"color", DANGER_COLOR,
				"background", LIGHT_BACKGROUND,
			)...,
		)
	}

	return nil
}

// boxHoverEffect creates hover background style for a selector.
func boxHoverEffect(tag string, isDark bool) *html.CSSContent {
	if isDark {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"background", DARK_BACKGROUND_HOVER,
			)...,
		)
	} else if !isDark {
		return html.NewCSSContent(tag,
			html.NewAttributes(
				"background", LIGHT_BACKGROUND_HOVER,
			)...,
		)
	}

	return nil
}
