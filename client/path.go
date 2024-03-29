package client

import (
	"path"
	"strings"
)

// prefixes
const (
	TranscoderPrefix     = "service/transcode/"
	SourcePrefix         = "service/source/"
	FanoutPrefix         = "service/fanout/"
	StreamPrefix         = "stream/"
	StreamSettingsPrefix = "streamSettings/"
	servicePrefix        = "service/"
)

// ParseServiceName parses service name from path, returns "" if path is not a service path
func ParseServiceName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return ""
	}

	return parts[2]
}

func ParseStreamName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return ""
	}

	return parts[1]
}

func ParseStreamTranscoder(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return ""
	}

	return parts[2]
}

func PathIsStream(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] != "stream" {
		return false
	}
	return true
}

func PathIsStreamTranscoder(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "stream" || parts[2] != "transcoder" {
		return false
	}
	return true
}

func PathIsStreamSettings(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "stream" || parts[2] != "settings" {
		return false
	}
	return true
}

func ServicePath(serviceName string, clientName string) string {
	return path.Join(servicePrefix, serviceName, clientName)
}

func StreamPath(name string) string {
	return path.Join(StreamPrefix, name)
}

func StreamTranscoderPath(name string) string {
	return path.Join(StreamPrefix, name, "transcoder")
}

func StreamSettingsPath(name string) string {
	return path.Join(StreamSettingsPrefix, name)
}

func ServicePrefix(prefix string) string {
	return path.Join(servicePrefix, prefix)
}
