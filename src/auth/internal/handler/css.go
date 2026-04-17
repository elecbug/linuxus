package handler

import (
	"fmt"

	"github.com/elecbug/linuxus/src/auth/internal/page"
)

// getLoginCSS builds CSS used by the login page.
func getLoginCSS() *page.CSS {
	return page.NewCSS().AddContents(
		loginBodyCSS(),
		loginTooltipCSS(),
		loginErrorCSS(),
	).AddContents(
		footerCSS(40)...,
	).AddContents(
		loginFormCSS()...,
	).AddContents(
		loginButtonCSS()...,
	)
}

// getErrorCSS builds CSS used by the error page.
func getErrorCSS() *page.CSS {
	return page.NewCSS().AddContents(
		loginBodyCSS(),
		loginTooltipCSS(),
		loginErrorCSS(),
	).AddContents(
		footerCSS(40)...,
	)
}

// getServiceCSS builds CSS used by the service shell page.
func getServiceCSS() *page.CSS {
	return page.NewCSS().AddContents(
		serviceBodyCSS(),
	).AddContents(
		serviceToorBarCSS()...,
	).AddContents(
		serviceIframeCSS()...,
	).AddContents(
		serviceButtonCSS()...,
	).AddContents(
		footerCSS(20)...,
	)

}

// footerCSS returns footer style rules with a configurable top margin.
func footerCSS(marginTop int) []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("footer",
			page.NewAttributes(
				"margin-top", fmt.Sprintf("%dpx", marginTop),
				"margin-bottom", "20px",
				"font-size", "0.9em",
				"text-align", "center",
				"color", FOOTER_COLOR,
			)...,
		),
		page.NewCSSContent("footer a",
			page.NewAttributes(
				"color", FOOTER_LINK_COLOR,
				"text-decoration", "none",
			)...,
		),
		page.NewCSSContent("footer a:hover",
			page.NewAttributes(
				"color", FOOTER_LINK_HOVER_COLOR,
				"text-decoration", "underline",
			)...,
		),
		page.NewCSSContent("footer a:active",
			page.NewAttributes(
				"color", FOOTER_LINK_ACTIVE_COLOR,
				"text-decoration", "underline",
			)...,
		),
		page.NewCSSContent("footer a:visited",
			page.NewAttributes(
				"color", FOOTER_LINK_VISITED_COLOR,
				"text-decoration", "underline",
			)...,
		),
	}
}

// loginBodyCSS returns base layout styles for login and error pages.
func loginBodyCSS() *page.CSSContent {
	return page.NewCSSContent("body",
		page.NewAttributes(
			"font-family", BASE_FONT_FAMILY,
			"max-width", "420px",
			"margin", "60px auto",
			"background", BASE_BACKGROUND,
			"color", BASE_COLOR,
		)...,
	)
}

// loginFormCSS returns style rules for the login form layout.
func loginFormCSS() []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("form.login-form",
			page.NewAttributes(
				"display", "flex",
				"flex-direction", "column",
				"gap", "12px",
			)...,
		),
	}
}

// loginErrorCSS returns style rules for login error messages.
func loginErrorCSS() *page.CSSContent {
	return page.NewCSSContent("p.error",
		page.NewAttributes(
			"color", DANGER_COLOR,
		)...,
	)
}

// loginTooltipCSS returns style rules for helper text under the form.
func loginTooltipCSS() *page.CSSContent {
	return page.NewCSSContent("p.tooltip",
		page.NewAttributes(
			"color", TOOLTIP_COLOR,
			"font-size", "0.9em",
			"font-style", "italic",
		)...,
	)
}

// loginButtonCSS returns style rules for login input and button elements.
func loginButtonCSS() []*page.CSSContent {
	return []*page.CSSContent{
		box("input", false, false),
		box("button", true, false),
		boxHoverEffect("button:hover", true),
	}
}

// serviceBodyCSS returns base layout styles for the service page.
func serviceBodyCSS() *page.CSSContent {
	return page.NewCSSContent("html, body",
		page.NewAttributes(
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

// serviceToolBarCSS returns style rules for the service top toolbar.
func serviceToolBarCSS() []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("div.topbar",
			page.NewAttributes(
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
		page.NewCSSContent("div.left",
			page.NewAttributes(
				"font-size", "1.2em",
				"font-weight", "bold",
			)...,
		),
		page.NewCSSContent("div.right",
			page.NewAttributes(
				"display", "flex",
				"gap", "10px",
			)...,
		),
	}
}

// serviceIframeCSS returns style rules for the terminal iframe area.
func serviceIframeCSS() []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("div.frame-wrap",
			page.NewAttributes(
				"flex", "1",
				"min-height", "0",
			)...,
		),
		page.NewCSSContent("iframe",
			page.NewAttributes(
				"width", "100%",
				"height", "100%",
				"border", "0",
				"display", "block",
			)...,
		),
	}
}

// serviceButtonCSS returns style rules for service action buttons.
func serviceButtonCSS() []*page.CSSContent {
	return []*page.CSSContent{
		box("a.btn", true, false),
		boxHoverEffect("a.btn:hover", true),
		box("a.btn-danger", true, true),
		boxHoverEffect("a.btn-danger:hover", true),
	}
}

// box creates a reusable button/input style rule variant.
func box(tag string, isDark, isDangerOpt bool) *page.CSSContent {
	if isDark && !isDangerOpt {
		return page.NewCSSContent(tag,
			page.NewAttributes(
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
		return page.NewCSSContent(tag,
			page.NewAttributes(
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
		return page.NewCSSContent(tag,
			page.NewAttributes(
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
		return page.NewCSSContent(tag,
			page.NewAttributes(
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

// boxHoverEffect creates hover style rules for reusable button variants.
func boxHoverEffect(tag string, isDark bool) *page.CSSContent {
	if isDark {
		return page.NewCSSContent(tag,
			page.NewAttributes(
				"background", DARK_BACKGROUND_HOVER,
			)...,
		)
	} else if !isDark {
		return page.NewCSSContent(tag,
			page.NewAttributes(
				"background", LIGHT_BACKGROUND_HOVER,
			)...,
		)
	}

	return nil
}
