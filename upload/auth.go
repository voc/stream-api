package upload

import (
	"io/ioutil"
	"log"

	"github.com/minio/pkg/wildcard"
	"github.com/pelletier/go-toml"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	// Auth checks whether the user has access to the passed relative path
	// Note: doesn't check whether the path is actually relative
	Auth(user string, pass string, path string) bool
}

type AuthConfigEntry struct {
	Match string `toml:"match"`
	User  string `toml:"user"`
	Pass  string `toml:"pass"`
}

type AuthConfig struct {
	Auth []AuthConfigEntry
}

type StaticAuth struct {
	conf map[string]AuthConfigEntry
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
		conf: make(map[string]AuthConfigEntry),
	}
	for _, entry := range conf.Auth {
		a.conf[entry.User] = entry
	}
	return a, nil
}

func (a *StaticAuth) Auth(user string, pass string, path string) bool {
	entry, ok := a.conf[user]
	if !ok {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(entry.Pass), []byte(pass))
	if err != nil {
		return false
	}
	return wildcard.MatchSimple(entry.Match, path)
}
