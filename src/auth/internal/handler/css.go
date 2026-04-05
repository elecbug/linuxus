package handler

import "github.com/elecbug/linuxus/src/auth/internal/page"

func (a *App) GetLoginCSS() *page.CSS {
	return page.NewCSS(
		page.NewCSSContent("body",
			page.KeyValue{Key: "font-family", Value: "sans-serif"},
			page.KeyValue{Key: "max-width", Value: "420px"},
			page.KeyValue{Key: "margin", Value: "60px auto"},
			page.KeyValue{Key: "background", Value: "#2e2e2e"},
			page.KeyValue{Key: "color", Value: "white"},
		),
		page.NewCSSContent("form",
			page.KeyValue{Key: "display", Value: "flex"},
			page.KeyValue{Key: "flex-direction", Value: "column"},
			page.KeyValue{Key: "gap", Value: "12px"},
		),
		baseBox("input", false, false),
		baseBox("button", true, false),
		hoverEffect("button:hover", true),
		page.NewCSSContent(".error",
			page.KeyValue{Key: "color", Value: "#ff0000"},
		),
	)
}

func (a *App) GetServiceCSS() string {
	return `
	<style>
        html, body {
            margin: 0;
            padding: 0;
            height: 100%;
            font-family: sans-serif;
        }
        .topbar {
            height: 56px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 16px;
            box-sizing: border-box;
            border-bottom: 1px solid #dddddd;
            background: #595959;
			color: white;
        }
        .left {
            font-weight: bold;
        }
        .right {
            display: flex;
            gap: 10px;
        }
        .frame-wrap {
            height: calc(100% - 56px);
        }
        iframe {
            width: 100%;
            height: 100%;
            border: 0;
            display: block;
        }
        </style>
        `
	// .btn {` + baseBox(3, true, false) + `}
	// .btn:hover {` + hoverEffect(3, true) + `}
	// .btn-danger {` + baseBox(3, true, true) + `}
	// .btn-danger:hover {` + hoverEffect(3, true) + `}
}

func baseBox(tag string, isDark, isDangerOpt bool) *page.CSSContent {
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

func hoverEffect(tag string, isDark bool) *page.CSSContent {
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
