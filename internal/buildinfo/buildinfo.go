package buildinfo

import "os"

// Version is overridden by -ldflags in release builds.
var Version = "dev"

// WebVersion may be provided when the frontend is built or shipped separately.
var WebVersion = ""

func RuntimeVersion() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	return Version
}

func RuntimeWebVersion() string {
	if v := os.Getenv("WEB_APP_VERSION"); v != "" {
		return v
	}
	return WebVersion
}
