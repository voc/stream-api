from flask import Blueprint

from api import db
import api.output


bp = Blueprint("transcoder", __name__, url_prefix="/transcoder")

class Transcoder(db.Model):
    __tablename__ = 'transcoders'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False, unique=True)
    title = db.Column(db.String(255), nullable=False)
    lastUpdated = db.Column(db.Date)


    def __init__(self, title=None, year=None):
        self.title = title
        self.year = year

    def __repr__(self):
        return "Movie(%r, %r, %r)" % (self.title, self.year, self.director)



@bp.route("/hello", methods=("POST",))
def transcoder_hello():
    # update transcoder age in db
    # update transcodings?
    db.output_state()
    api.output.render_state("state.json")


    # find/add transcoder
    # update transcoder age


# timeout old transcoders
def timeout_transcoders():
