from api import db
from api.stream import Stream

#{"streams": [{"key": "sloop", "source": "ingest.c3voc.de:8000", "artwork": {"base": "http://live.ber.c3voc.de/artwork"}, "transcoding": {"worker": "loop-transcoder", "sink": "live.ber.c3voc.de:7999"}, "lastUpdated": 1582568701}]}

class Stream(db.Model):
    __tablename__ = 'streams'

    id = db.Column(db.Integer, primary_key=True)

    transcoded_by = db.Column(db.Integer, ForeignKey('transcoders.id')
)    transcoder = db.relation("Transcoder", backref='streams', lazy=False)


    source = db.Column(db.String(100))
    key = db.Column(db.String(50))

    # only allow each key once per source
    db.UniqueConstraint(source, key, name='unique_stream')

    lastUpdated = db.Column(db.Date)

    def __init__(self, name=None):
        self.name = name

    def __repr__(self):
        return "Director(%r)" % (self.name)


# Map Icecast stream source
def map_icecast(source, backend):
    url = source["listenurl"]
    return url.split("/")[-1]

# Get streams from icecast2
def fetch_icecast(backend):
    sources = []
    url = f"http://{backend['address']}/status-json.xsl"

    try:
        f = urllib.request.urlopen(url)
        result = json.load(f)["icestats"]

        # check whether streams exist at all
        if not "source" in result:
            return []

        # one stream
        if isinstance(result["source"], dict):
            sources.append(result["source"])

        # multiple streams
        elif isinstance(result["source"], list):
            sources = result["source"]

        #print("icecast", url, streams, "\n")
        sources = [map_icecast(source, backend) for source in sources]

    except URLError:
        print("Error: Could not fetch from", url)
        return []

    return sources

# add/update source streams
def update_streams():

# timeout old source streams

def timeout_streams():
