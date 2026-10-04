package config

import "testing"

func TestRejectInvalidOperationalSettings(t *testing.T) {
	for _, change := range []func(*Config){
		func(c *Config) { c.Capacity.MaxPending = -1 }, func(c *Config) { c.Capacity.MinFreeSpace = "invalid" },
		func(c *Config) { c.UserService.Templates = `{"python":{"image":"image","sead":"/opt"}}` },
		func(c *Config) { c.UserService.Templates = `{"default":{"image":"image"}}` },
		func(c *Config) { c.UserService.Classes = `{"class-a":"unknown"}` },
		func(c *Config) { c.UserService.Templates = `{"python":{"image":"image","seed":"../outside"}}` },
		func(c *Config) { c.UserService.Templates = `{} {}` },
	} {
		var cfg Config
		change(&cfg)
		if err := validateOperations(&cfg); err == nil {
			t.Fatal("invalid operational settings accepted")
		}
	}
}
