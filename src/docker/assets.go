// Package docker contains the Docker resources shipped with linuxusctl.
package docker

import "embed"

// Files contains runtime image definitions, the startup script, and classroom seeds.
//
//go:embed *.Dockerfile start.sh templates
var Files embed.FS
