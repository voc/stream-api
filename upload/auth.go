package upload

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/pkg/wildcard"
)

type Auth interface {
	// Auth checks whether the user has access to the passed relative path
	// Note: doesn't check whether the path is actually relative
	Auth(user string, pass string, path string) (string, bool)
}

type AuthConfigEntry struct {
	Match string `toml:"match"`
	User  string `toml:"user"`
	Pass  string `toml:"pass"`
}

type AuthConfig struct {
	AllowedDirs []string `toml:"allowedDirs"`
	Users       []AuthConfigEntry
}

type StaticAuth struct {
	allowedDirs []string
	conf        map[string]AuthConfigEntry
}

func NewStaticAuth(conf AuthConfig) Auth {
	a := &StaticAuth{
		conf: make(map[string]AuthConfigEntry),
	}
	for _, dir := range conf.AllowedDirs {
		if dir[len(dir)-1] != os.PathSeparator {
			dir += string(os.PathSeparator)
		}
		a.allowedDirs = append(a.allowedDirs, dir)
	}
	for _, entry := range conf.Users {
		a.conf[entry.User] = entry
	}
	return a
}

func (a *StaticAuth) Auth(user string, pass string, path string) (string, bool) {
	entry, ok := a.conf[user]
	if !ok {
		return "", false
	}
	if pass != entry.Pass {
		return "", false
	}

	// split path
	cleanedPath := filepath.Clean(path)

	// check path for allowed prefixes
	found := false
	for _, prefix := range a.allowedDirs {
		if strings.HasPrefix(cleanedPath, prefix) {
			found = true
			cleanedPath = cleanedPath[len(prefix):]
			break
		}
	}
	if !found {
		return "", false
	}
	if !filepath.IsAbs(cleanedPath) {
		cleanedPath = string(os.PathSeparator) + cleanedPath
	}

	// determine slug
	parts := strings.Split(cleanedPath, string(os.PathSeparator))
	slug := parts[1]

	// match against entry
	if !wildcard.MatchSimple(entry.Match, slug) {
		return "", false
	}

	return slug, true
}
