from flask import Flask
from app.api import views as api_views
from app.auth import views as auth_views
from app import manage
from app.extensions import apispec
from app.extensions import db
from app.extensions import jwt
from app.extensions import migrate, celery
from dotenv import load_dotenv


def create_app(testing=False):
    """Application factory, used to create application"""
    load_dotenv('.env')
    app = Flask("app")
    app.config.from_object("app.config")

    if testing is True:
        app.config["TESTING"] = True
    
    
    configure_extensions(app)
    configure_cli(app)
    configure_apispec(app)
    register_blueprints(app)
    init_celery(app)
    with app.app_context():
        api_views.register_views()
        auth_views.register_views()
    return app


def configure_extensions(app):
    """Configure flask extensions"""
    db.init_app(app)
    jwt.init_app(app)
    migrate.init_app(app, db)


def configure_cli(app):
    """Configure Flask 2.0's cli for easy entity management"""
    app.cli.add_command(manage.init)


def configure_apispec(app):
    """Configure APISpec for swagger support"""
    apispec.init_app(app, security=[{"jwt": []}])
    apispec.spec.components.security_scheme(
        "jwt", {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}
    )
    apispec.spec.components.schema(
        "PaginatedResult",
        {
            "properties": {
                "total": {"type": "integer"},
                "pages": {"type": "integer"},
                "next": {"type": "string"},
                "prev": {"type": "string"},
            }
        },
    )


def register_blueprints(app):
    """Register all blueprints for application"""
    app.register_blueprint(auth_views.blueprint)
    app.register_blueprint(api_views.blueprint)


def init_celery(app=None):
    app = app or create_app()
    celery.conf.update(app.config.get("CELERY", {}))

    class ContextTask(celery.Task):
        """Make celery tasks work with Flask app context"""

        def __call__(self, *args, **kwargs):
            with app.app_context():
                return self.run(*args, **kwargs)

    celery.Task = ContextTask
    return celery
