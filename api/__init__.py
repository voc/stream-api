import os

from flask import Flask
from flask_sqlalchemy import SQLAlchemy

db = SQLAlchemy()

def create_app(test_config=None):
    app = Flask(__name__, instance_relative_config=True)

    db_url = "sqlite:///" + os.path.join(app.instance_path, "stream.sqlite")

    app.config.from_mapping(
        SQLALCHEMY_DATABASE_URI=db_url,
    )

    from api import transcoder

    app.register_blueprint(transcoder.bp)

    # make "index" point at "/", which is handled by "blog.index"
    app.add_url_rule("/", endpoint="index")

    db = SQLAlchemy(app)

def init_db():
    db.drop_all()
    db.create_all()


class User(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(80), unique=True, nullable=False)
    email = db.Column(db.String(120), unique=True, nullable=False)

    def __repr__(self):
        return '<User %r>' % self.username

@app.route("/")
def ping():
    return "Pong"
