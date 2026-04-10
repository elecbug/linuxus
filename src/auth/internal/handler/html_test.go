package handler_test

import (
	"testing"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
)

func TestHTMLRendering(t *testing.T) {
	config := handler.AppConfig{
		Users:                   nil,
		SessionKey:              nil,
		LoginPath:               "login",
		LogoutPath:              "logout",
		ServicePath:             "service",
		TerminalPath:            "terminal",
		AdminUserID:             "adminContainer",
		UserContainerNamePrefix: "linuxus-user-",
		TrustedProxies:          nil,
	}

	app := handler.NewApp(config)
	loginPage := app.GetLoginPage()
	if loginPage == "" {
		t.Error("GetLoginPage returned an empty string")
	} else {
		t.Log("GetLoginPage output:")
		t.Log("\n" + loginPage)
	}
}

func TestGetErrorPageRendering(t *testing.T) {
	config := handler.AppConfig{
		Users:                   nil,
		SessionKey:              nil,
		LoginPath:               "login",
		LogoutPath:              "logout",
		ServicePath:             "service",
		TerminalPath:            "terminal",
		AdminUserID:             "adminContainer",
		UserContainerNamePrefix: "linuxus-user-",
		TrustedProxies:          nil,
	}

	app := handler.NewApp(config)
	errorPage := app.GetErrorPage()
	if errorPage == "" {
		t.Error("GetErrorPage returned an empty string")
	} else {
		t.Log("GetErrorPage output:")
		t.Log("\n" + errorPage)
	}
}
