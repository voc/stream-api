package stream

type Stream struct {
	Format string `json:"format"` // ffmpeg format descriptor
	Source string `json:"source"` // complete source URL
	Slug   string `json:"slug"`   // stream slug
}

type GlobalConfig struct {
	IcecastUser     string `json:"icecastUser"`
	IcecastPassword string `json:"icecastPassword"`

	// TODO: replace this with a dynamic CDN-sink service
	Sink string `json:"sink"`
}
