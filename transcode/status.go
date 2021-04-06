package transcode

type TranscoderStatus struct {
	Name     string
	Capacity int
	Streams  []string
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// ByLoad implements sort.Interface for transcoders based on job load
type ByLoad []*TranscoderStatus

func (l ByLoad) Len() int      { return len(l) }
func (l ByLoad) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l ByLoad) Less(i, j int) bool {
	// If we have no capacity, order by count alone
	if l[i].Capacity <= 0 || l[j].Capacity <= 0 {
		return len(l[i].Streams) < len(l[j].Streams)
	}

	// Order based on load
	return float64(len(l[i].Streams))/float64(l[i].Capacity) <
		float64(len(l[j].Streams))/float64(l[j].Capacity)
}
