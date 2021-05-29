package source

import (
	"reflect"
	"testing"
)

func Test_icecastSource(t *testing.T) {
	data := `{"icestats":{"admin":"contact@c3voc.de","host":"ingest.c3voc.de","location":"voc","server_id":"Icecast 2","server_start":"Wed, 04 Nov 2020 12:10:09 +0100","server_start_iso8601":"2020-11-04T12:10:09+0100","source":{"genre":"various","listener_peak":2,"listeners":1,"listenurl":"http://ingest.c3voc.de:8000/q1","server_description":"Unspecified description","server_name":"Unspecified name","server_type":"video/webm","stream_start":"Sun, 08 Nov 2020 00:11:41 +0100","stream_start_iso8601":"2020-11-08T00:11:41+0100","dummy":null}}}`

	var source IcecastSource

	iceStreams, err := source.parse([]byte(data))
	if err != nil {
		t.Error(err)
	}
	streams := source.mapStreams(iceStreams)

	expected := []Stream{
		{Format: "matroska", Source: "http://ingest.c3voc.de:8000/q1", Slug: "q1"},
	}
	if !reflect.DeepEqual(streams, expected) {
		t.Errorf("Got streams %v, expected %v", streams, expected)
	}
}
