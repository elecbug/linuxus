package handler

import (
	"testing"
)

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
	loginPage := GetLoginPage(app)
	if loginPage == "" {
		t.Error("GetLoginPage returned an empty string")
	} else {
		t.Log("GetLoginPage output:")
		t.Log("\n" + loginPage)
	}
}

func TestGetErrorPageRendering(t *testing.T) {
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
	errorPage := GetErrorPage(app)
	if errorPage == "" {
		t.Error("GetErrorPage returned an empty string")
	} else {
		t.Log("GetErrorPage output:")
		t.Log("\n" + errorPage)
	}
}
