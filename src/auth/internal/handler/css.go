package handler

import (
	"fmt"

	"github.com/elecbug/linuxus/src/auth/internal/page"
)

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

func getErrorCSS() *page.CSS {
	return page.NewCSS().AddContents(
		loginBodyCSS(),
		loginTooltipCSS(),
		loginErrorCSS(),
	).AddContents(
		footerCSS(40)...,
	)
}

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

func footerCSS(marginTop int) []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("footer",
			page.NewAttributes(
				"margin-top", fmt.Sprintf("%dpx", marginTop),
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

func loginErrorCSS() *page.CSSContent {
	return page.NewCSSContent("p.error",
		page.NewAttributes(
			"color", DANGER_COLOR,
		)...,
	)
}

func loginTooltipCSS() *page.CSSContent {
	return page.NewCSSContent("p.tooltip",
		page.NewAttributes(
			"color", TOOLTIP_COLOR,
			"font-size", "0.9em",
			"font-style", "italic",
		)...,
	)
}

func loginButtonCSS() []*page.CSSContent {
	return []*page.CSSContent{
		box("input", false, false),
		box("button", true, false),
		boxHoverEffect("button:hover", true),
	}
}

func serviceBodyCSS() *page.CSSContent {
	return page.NewCSSContent("html, body",
		page.NewAttributes(
			"margin", "0",
			"padding", "0",
			"height", "100%",
			"font-family", BASE_FONT_FAMILY,
			"background", BASE_BACKGROUND,
			"color", BASE_COLOR,
		)...,
	)
}

func serviceToorBarCSS() []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("div.topbar",
			page.NewAttributes(
				"height", "56px",
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

func serviceIframeCSS() []*page.CSSContent {
	return []*page.CSSContent{
		page.NewCSSContent("div.frame-wrap",
			page.NewAttributes(
				"height", "calc(100% - 56px - 50px)",
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

func serviceButtonCSS() []*page.CSSContent {
	return []*page.CSSContent{
		box("a.btn", true, false),
		boxHoverEffect("a.btn:hover", true),
		box("a.btn-danger", true, true),
		boxHoverEffect("a.btn-danger:hover", true),
	}
}

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
