package stream

type Stream struct {
	Format      string `json:"format"`      // ffmpeg format descriptor
	Source      string `json:"source"`      // complete source URL
	Slug        string `json:"slug"`        // stream slug
	PublishedAt int    `json:"publishedAt"` // publish timestamp in unix format
}

type StreamOptions struct {
	Passthrough bool `json:"passthrough"`
	// audio only?
}

type Settings struct {
	Slug       string        `json:"slug"`       // stream slug
	IngestType string        `json:"ingestType"` // mode: ingest vs. relay
	Secret     string        `json:"secret"`     // stream secret for authentication
	Public     bool          `json:"public"`     // whether the stream should be available publically
	Options    StreamOptions `json:"options"`    // additional stream options
}

type GlobalConfig struct {
	IcecastUser     string `json:"icecastUser"`
	IcecastPassword string `json:"icecastPassword"`

	// TODO: replace this with a dynamic CDN-sink service
	Sink string `json:"sink"`
}
