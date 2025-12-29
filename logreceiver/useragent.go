package logreceiver

import "strings"

// ClientType represents the anonymized streaming client type.
type ClientType string

const (
	ClientTypeChrome  ClientType = "chrome"
	ClientTypeFirefox ClientType = "firefox"
	ClientTypeSafari  ClientType = "safari"
	ClientTypeMPV     ClientType = "mpv"
	ClientTypeVLC     ClientType = "vlc"
	ClientTypeFFmpeg  ClientType = "ffmpeg"
	ClientTypeOther   ClientType = "other"
)

// ParseUserAgent returns the anonymized client type based on the user agent string.
func ParseUserAgent(userAgent string) ClientType {
	if userAgent == "" {
		return ClientTypeOther
	}
	if strings.Contains(userAgent, "Chrome") {
		return ClientTypeChrome
	}
	if strings.Contains(userAgent, "Firefox") {
		return ClientTypeFirefox
	}
	if strings.Contains(userAgent, "libmpv") {
		return ClientTypeMPV
	}
	if strings.Contains(userAgent, "VLC") {
		return ClientTypeVLC
	}
	if strings.Contains(userAgent, "Lavf") {
		return ClientTypeFFmpeg
	}
	// check for safari last, because other HLS clients might add "Safari" to their user agent
	if strings.Contains(userAgent, "Safari") {
		return ClientTypeSafari
	}

	return ClientTypeOther
}
