package handler

import "github.com/elecbug/linuxus/src/auth/internal/page"

func getLoginCSS() *page.CSS {
	return page.NewCSS(
		page.NewCSSContent("body",
			page.KeyValue{Key: "font-family", Value: BASE_FONT_FAMILY},
			page.KeyValue{Key: "max-width", Value: "420px"},
			page.KeyValue{Key: "margin", Value: "60px auto"},
			page.KeyValue{Key: "background", Value: BASE_BACKGROUND},
			page.KeyValue{Key: "color", Value: BASE_COLOR},
		),
		page.NewCSSContent("form",
			page.KeyValue{Key: "display", Value: "flex"},
			page.KeyValue{Key: "flex-direction", Value: "column"},
			page.KeyValue{Key: "gap", Value: "12px"},
		),
		box("input", false, false),
		box("button", true, false),
		boxHoverEffect("button:hover", true),
		page.NewCSSContent(".error",
			page.KeyValue{Key: "color", Value: DANGER_COLOR},
		),
	)
}

func getServiceCSS() *page.CSS {
	return page.NewCSS(
		page.NewCSSContent("html, body",
			page.KeyValue{Key: "margin", Value: "0"},
			page.KeyValue{Key: "padding", Value: "0"},
			page.KeyValue{Key: "height", Value: "100%"},
			page.KeyValue{Key: "font-family", Value: BASE_FONT_FAMILY},
		),
		page.NewCSSContent(".topbar",
			page.KeyValue{Key: "height", Value: "56px"},
			page.KeyValue{Key: "display", Value: "flex"},
			page.KeyValue{Key: "align-items", Value: "center"},
			page.KeyValue{Key: "justify-content", Value: "space-between"},
			page.KeyValue{Key: "padding", Value: "0 16px"},
			page.KeyValue{Key: "box-sizing", Value: "border-box"},
			page.KeyValue{Key: "border-bottom", Value: DARK_BORDER},
			page.KeyValue{Key: "background", Value: BASE_BACKGROUND},
			page.KeyValue{Key: "color", Value: BASE_COLOR},
		),
		page.NewCSSContent(".left",
			page.KeyValue{Key: "font-weight", Value: "bold"},
		),
		page.NewCSSContent(".right",
			page.KeyValue{Key: "display", Value: "flex"},
			page.KeyValue{Key: "gap", Value: "10px"},
		),
		page.NewCSSContent(".frame-wrap",
			page.KeyValue{Key: "height", Value: "calc(100% - 56px)"},
		),
		page.NewCSSContent("iframe",
			page.KeyValue{Key: "width", Value: "100%"},
			page.KeyValue{Key: "height", Value: "100%"},
			page.KeyValue{Key: "border", Value: "0"},
			page.KeyValue{Key: "display", Value: "block"},
		),
		box(".btn", true, false),
		boxHoverEffect(".btn:hover", true),
		box(".btn-danger", true, true),
		boxHoverEffect(".btn-danger:hover", true),
	)

}

func box(tag string, isDark, isDangerOpt bool) *page.CSSContent {
	if isDark && !isDangerOpt {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "display", Value: BOX_DISPLAY},
			page.KeyValue{Key: "padding", Value: BOX_PADDING},
			page.KeyValue{Key: "text-decoration", Value: BOX_TEXT_DECORATION},
			page.KeyValue{Key: "border", Value: DARK_BORDER},
			page.KeyValue{Key: "border-radius", Value: BOX_BORDER_RADIUS},
			page.KeyValue{Key: "font-size", Value: BOX_FONT_SIZE},
			page.KeyValue{Key: "color", Value: DARK_COLOR},
			page.KeyValue{Key: "background", Value: DARK_BACKGROUND},
		)
	} else if !isDark && !isDangerOpt {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "display", Value: BOX_DISPLAY},
			page.KeyValue{Key: "padding", Value: BOX_PADDING},
			page.KeyValue{Key: "text-decoration", Value: BOX_TEXT_DECORATION},
			page.KeyValue{Key: "border", Value: LIGHT_BORDER},
			page.KeyValue{Key: "border-radius", Value: BOX_BORDER_RADIUS},
			page.KeyValue{Key: "font-size", Value: BOX_FONT_SIZE},
			page.KeyValue{Key: "color", Value: LIGHT_COLOR},
			page.KeyValue{Key: "background", Value: LIGHT_BACKGROUND},
		)
	} else if isDark && isDangerOpt {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "display", Value: BOX_DISPLAY},
			page.KeyValue{Key: "padding", Value: BOX_PADDING},
			page.KeyValue{Key: "text-decoration", Value: BOX_TEXT_DECORATION},
			page.KeyValue{Key: "border", Value: DANGER_BORDER},
			page.KeyValue{Key: "border-radius", Value: BOX_BORDER_RADIUS},
			page.KeyValue{Key: "font-size", Value: BOX_FONT_SIZE},
			page.KeyValue{Key: "color", Value: DANGER_COLOR},
			page.KeyValue{Key: "background", Value: DARK_BACKGROUND},
		)
	} else if !isDark && isDangerOpt {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "display", Value: BOX_DISPLAY},
			page.KeyValue{Key: "padding", Value: BOX_PADDING},
			page.KeyValue{Key: "text-decoration", Value: BOX_TEXT_DECORATION},
			page.KeyValue{Key: "border", Value: DANGER_BORDER},
			page.KeyValue{Key: "border-radius", Value: BOX_BORDER_RADIUS},
			page.KeyValue{Key: "font-size", Value: BOX_FONT_SIZE},
			page.KeyValue{Key: "color", Value: DANGER_COLOR},
			page.KeyValue{Key: "background", Value: LIGHT_BACKGROUND},
		)
	}

	return nil
}

func boxHoverEffect(tag string, isDark bool) *page.CSSContent {
	if isDark {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "background", Value: DARK_BACKGROUND_HOVER},
		)
	} else if !isDark {
		return page.NewCSSContent(tag,
			page.KeyValue{Key: "background", Value: LIGHT_BACKGROUND_HOVER},
		)
	}

	return nil
}
