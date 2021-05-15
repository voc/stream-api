package monitor

import "embed"

// content holds our static web server content.
//go:embed frontend/public
var static embed.FS
