// Package static contains the web assets shipped with the application.
package static

import "embed"

// Files contains the authentication server's static assets.
//
//go:embed favicon.png
var Files embed.FS
