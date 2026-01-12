package transcode

// TranscoderStatus represents the transcoder state as announced via etcd
type TranscoderStatus struct {
	Name       string `json:"name"`
	Capacity   int    `json:"capacity"`
	NumStreams int    `json:"streams"`
}

// ByLoad implements sort.Interface for transcoders based on job load
type ByLoad []*TranscoderStatus

func (l ByLoad) Len() int      { return len(l) }
func (l ByLoad) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l ByLoad) Less(i, j int) bool {
	// If we have no capacity, order by count alone
	if l[i].Capacity <= 0 || l[j].Capacity <= 0 {
		return l[i].NumStreams < l[j].NumStreams
	}

	// Order based on load
	return float64(l[i].NumStreams)/float64(l[i].Capacity) <
		float64(l[j].NumStreams)/float64(l[j].Capacity)
}
