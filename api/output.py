
#{"streams": [{"key": "sloop", "source": "ingest.c3voc.de:8000", "artwork": {"base": "http://live.ber.c3voc.de/artwork"}, "transcoding": {"worker": "loop-transcoder", "sink": "live.ber.c3voc.de:7999"}, "lastUpdated": 1582568701}]}

def render_state():
	# get streams
	# output to json
	# atomically replace