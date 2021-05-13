package fanout

type FanoutStatus struct {
	Name    string   `json:"name"`
	Sink    string   `json:"sink"`
	Streams []string `json:"streams"`
}
