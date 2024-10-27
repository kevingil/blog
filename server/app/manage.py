import click
from flask.cli import with_appcontext


@click.command("init")
@with_appcontext
def init():
    """Create a new admin user"""
    from app.extensions import db
    from app.models import User

    click.echo("create user")
    user = User(name="admin", email="admin@admin.com", password="admin123", active=True)
    db.session.add(user)
    db.session.commit()
    click.echo("created user admin")
