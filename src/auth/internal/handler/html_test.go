package handler_test

import (
	"testing"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
)

func TestHTMLRendering(t *testing.T) {
	app := handler.NewApp(nil, nil, "login", "logout", "service", "terminal", "adminID", "adminPassword", "adminContainer")
	loginPage := app.GetLoginPage()
	if loginPage == "" {
		t.Error("GetLoginPage returned an empty string")
	} else {
		t.Log("GetLoginPage output:")
		t.Log("\n" + loginPage)
	}
}
