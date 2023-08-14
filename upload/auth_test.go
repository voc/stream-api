package upload

import "testing"

func TestStaticAuth(t *testing.T) {
	tests := []struct {
		name        string
		allowedDirs []string
		users       AuthConfigEntry
		username    string
		password    string
		path        string
		wantSlug    string
		wantOk      bool
	}{
		{"simple", []string{"/"}, AuthConfigEntry{"*", "user", "pass"}, "user", "pass", "/slug/test.mp4", "slug", true},
		{"invalidPass", []string{"/"}, AuthConfigEntry{"*", "user", "pass"}, "user", "invalid", "/slug/test.mp4", "", false},
		{"unknownUser", []string{"/"}, AuthConfigEntry{"*", "user", "pass"}, "user5", "pass", "/slug/test.mp4", "", false},
		{"subpathWithoutSlash", []string{"/allowed"}, AuthConfigEntry{"*", "user", "pass"}, "user", "pass", "/allowed/slug/test.mp4", "slug", true},
		{"subpathWithSlash", []string{"/allowed/"}, AuthConfigEntry{"*", "user", "pass"}, "user", "pass", "/allowed/slug/test.mp4", "slug", true},
		{"invalidPath", []string{"/allowed/"}, AuthConfigEntry{"*", "user", "pass"}, "user", "pass", "/slug/test.mp4", "", false},
		{"disallowIncompletePrefix", []string{"/allowed"}, AuthConfigEntry{"*", "user", "pass"}, "user", "pass", "/allowedfoo/slug/test.mp4", "", false},
		{"simpleMatch", []string{"/allowed"}, AuthConfigEntry{"slug", "user", "pass"}, "user", "pass", "/allowed/slug/test.mp4", "slug", true},
		{"invalidMatch", []string{"/allowed"}, AuthConfigEntry{"slug", "user", "pass"}, "user", "pass", "/allowed/invalid/test.mp4", "", false},
		{"wildcardMatch", []string{"/allowed"}, AuthConfigEntry{"slug*", "user", "pass"}, "user", "pass", "/allowed/slugallowed/test.mp4", "slugallowed", true},
		{"invalidWildcardMatch", []string{"/allowed"}, AuthConfigEntry{"slug*", "user", "pass"}, "user", "pass", "/allowed/invalidslug/test.mp4", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewStaticAuth(AuthConfig{
				AllowedDirs: tt.allowedDirs,
				Users:       []AuthConfigEntry{tt.users},
			})
			gotSlug, gotOk := a.Auth(tt.username, tt.password, tt.path)
			if gotOk != tt.wantOk {
				t.Errorf("StaticAuth.Auth() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotSlug != tt.wantSlug {
				t.Errorf("StaticAuth.Auth() slug = %v, want %v", gotSlug, tt.wantSlug)
			}
		})
	}
}

func TestAuth(t *testing.T) {
	path := "/hls/s2/segment_SD108.ts"
	username := "uploader"
	password := "uploader"

	a := NewStaticAuth(AuthConfig{
		AllowedDirs: []string{"/hls/"},
		Users:       []AuthConfigEntry{{"*", "uploader", "uploader"}},
	})

	gotSlug, gotOk := a.Auth(username, password, path)
	if gotOk != true {
		t.Errorf("StaticAuth.Auth() ok = %v, want %v", gotOk, true)
	}

	if gotSlug != "s2" {
		t.Errorf("StaticAuth.Auth() slug = %v, want %v", gotSlug, "s2")
	}
}
