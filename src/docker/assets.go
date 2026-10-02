// Package docker contains the Docker resources shipped with linuxusctl.
package docker

import "embed"

// Files contains only runtime image definitions and the user startup script.
//
//go:embed *.Dockerfile start.sh
var Files embed.FS
