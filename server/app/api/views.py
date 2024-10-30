from flask import Blueprint, current_app, jsonify
from flask_restful import Api
from pydantic import ValidationError
from app.extensions import apispec
from app.api.resources import UserResource, UserList

blueprint = Blueprint("api", __name__, url_prefix="/api/v1")
api = Api(blueprint)

api.add_resource(UserResource, "/users/<int:user_id>", endpoint="user_by_id")
api.add_resource(UserList, "/users", endpoint="users")

def register_views():
    # Register the paths
    apispec.spec.path(view=UserResource, app=current_app)
    apispec.spec.path(view=UserList, app=current_app)

@blueprint.errorhandler(ValidationError)
def handle_pydantic_error(e):
    """Return json error for pydantic validation errors.
    
    This will avoid having to try/catch ValidationErrors in all endpoints, returning
    correct JSON response with associated HTTP 400 Status (https://tools.ietf.org/html/rfc7231#section-6.5.1)
    """
    return jsonify({"errors": e.errors()}), 400
