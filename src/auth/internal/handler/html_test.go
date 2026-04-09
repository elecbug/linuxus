package handler_test

import (
	"testing"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
)

func TestHTMLRendering(t *testing.T) {
	app := handler.NewApp(
		nil, nil,
		"login", "logout", "service", "terminal",
		"adminContainer", "linuxus-user-",
		nil,
	)
	loginPage := app.GetLoginPage()
	if loginPage == "" {
		t.Error("GetLoginPage returned an empty string")
	} else {
		t.Log("GetLoginPage output:")
		t.Log("\n" + loginPage)
	}
}

func TestGetErrorPageRendering(t *testing.T) {
	app := handler.NewApp(
		nil, nil,
		"login", "logout", "service", "terminal",
		"adminContainer", "linuxus-user-",
		nil,
	)
	errorPage := app.GetErrorPage()
	if errorPage == "" {
		t.Error("GetErrorPage returned an empty string")
	} else {
		t.Log("GetErrorPage output:")
		t.Log("\n" + errorPage)
	}
}
