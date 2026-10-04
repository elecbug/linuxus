package config

import "testing"

func TestAuthRouteValidation(t *testing.T) {
	for _, tc := range []struct {
		path  string
		valid bool
	}{
		{"login", true}, {"auth/login-v2", true}, {"login.v2", true},
		{"", false}, {"/login", false}, {"login/", false}, {"auth//login", false},
		{"../login", false}, {"auth/../login", false}, {"login?query=1", false},
		{"login%2Flogout", false}, {"login#fragment", false}, {"{user}", false},
		{"bad path", false}, {"logout", false}, {"static", false}, {"static/custom", false}, {"favicon.ico", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var cfg Config
			cfg.AuthService.ServiceURL.Login = tc.path
			cfg.AuthService.ServiceURL.Logout = "logout"
			cfg.AuthService.ServiceURL.Service = "service"
			cfg.AuthService.ServiceURL.Terminal = "terminal"
			cfg.AuthService.ServiceURL.Signup = "signup"
			if err := ValidateAuthRoutes(&cfg); (err == nil) != tc.valid {
				t.Fatalf("route=%q error=%v", tc.path, err)
			}
		})
	}
}
