package handler

import (
	"testing"
)

// TestHTMLRendering verifies login page template generation is non-empty.
func TestHTMLRendering(t *testing.T) {
	config := &AppConfig{
		Users:                   nil,
		SessionKey:              nil,
		LoginPath:               "login",
		LogoutPath:              "logout",
		ServicePath:             "service",
		TerminalPath:            "terminal",
		UserContainerNamePrefix: "linuxus-user-",
		TrustedProxies:          nil,
	}

	app := NewApp(config)
	loginPage := getLoginPage(app)
	if loginPage == "" {
		t.Error("GetLoginPage returned an empty string")
	} else {
		t.Log("GetLoginPage output:")
		t.Log("\n" + loginPage)
	}
}

// TestGetErrorPageRendering verifies error page template generation is non-empty.
func TestGetErrorPageRendering(t *testing.T) {
	errorPage := getErrorPage()
	if errorPage == "" {
		t.Error("GetErrorPage returned an empty string")
	} else {
		t.Log("GetErrorPage output:")
		t.Log("\n" + errorPage)
	}
}
