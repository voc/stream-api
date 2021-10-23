package upload

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/pkg/wildcard"
	"github.com/pelletier/go-toml"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	// Auth checks whether the user has access to the passed relative path
	// Note: doesn't check whether the path is actually relative
	Auth(user string, pass string, path string) (string, bool)
}

type GlobalConfig struct {
	AllowedDirs []string `toml:"allowedDirs"`
}

type AuthConfigEntry struct {
	Match string `toml:"match"`
	User  string `toml:"user"`
	Pass  string `toml:"pass"`
}

type AuthConfig struct {
	Global GlobalConfig
	Auth   []AuthConfigEntry
}

type StaticAuth struct {
	allowedDirs []string
	conf        map[string]AuthConfigEntry
}

func NewStaticAuth(configPath string) (Auth, error) {
	conf := AuthConfig{}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	err = toml.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}
	log.Println("read auth from", configPath)

	a := &StaticAuth{
		allowedDirs: conf.Global.AllowedDirs,
		conf:        make(map[string]AuthConfigEntry),
	}
	for _, entry := range conf.Auth {
		a.conf[entry.User] = entry
	}
	return a, nil
}

func (a *StaticAuth) Auth(user string, pass string, path string) (string, bool) {
	entry, ok := a.conf[user]
	if !ok {
		return "", false
	}
	err := bcrypt.CompareHashAndPassword([]byte(entry.Pass), []byte(pass))
	if err != nil {
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

	// match against entry
	if !wildcard.MatchSimple(entry.Match, cleanedPath) {
		return "", false
	}

	// determine slug
	parts := strings.Split(cleanedPath, string(os.PathSeparator))
	slug := parts[1]
	return slug, true
}
