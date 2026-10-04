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

func TestAdminRouteDefaultsAndConflicts(t *testing.T) {
	for _, tc := range []struct {
		admin, login string
		valid        bool
	}{
		{"", "login", true}, {"admin", "login", true}, {"ops/classroom", "auth/login", true},
		{"ops", "admin", true}, {"/ops", "login", false}, {"ops/", "login", false},
		{"ops//admin", "login", false}, {"ops?mode=admin", "login", false},
		{"ops/{user}", "login", false}, {"static", "login", false},
		{"static/admin", "login", false}, {"healthz", "login", false}, {"favicon.ico", "login", false},
		{"ops", "ops", false}, {"ops", "ops/login", false}, {"ops/admin", "ops", false},
		{"service/manage", "login", false}, {"admin", "admin/api/users", false},
	} {
		t.Run(tc.admin+"/"+tc.login, func(t *testing.T) {
			cfg, err := ParseEnv([]byte(DefaultEnv))
			if err != nil {
				t.Fatal(err)
			}
			cfg.AuthService.ServiceURL.Admin = tc.admin
			cfg.AuthService.ServiceURL.Login = tc.login
			if err := ValidateAuthRoutes(&cfg); (err == nil) != tc.valid {
				t.Fatalf("admin=%q login=%q error=%v", tc.admin, tc.login, err)
			}
			if tc.admin == "" && cfg.AdminRoute() != "admin" {
				t.Fatal("legacy configuration lost its default administrator route")
			}
		})
	}
}
