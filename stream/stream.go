package stream

type Stream struct {
	Format string // ffmpeg format descriptor
	Source string // complete source URL
	Slug   string // stream slug
}

type GlobalConfig struct {
	IcecastUser     string
	IcecastPassword string

	// TODO: replace this with a dynamic CDN-sink service
	Sink string
}
