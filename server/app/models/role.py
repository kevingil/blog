from sqlalchemy.orm import validates
from sqlalchemy.sql import func
from blog.extensions import db

class Role(db.Model):
    """Role model for managing user permissions."""
    __tablename__ = 'roles'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(80), unique=True, nullable=False, index=True)
    
    # Timestamps
    created_at = db.Column(db.DateTime, default=func.now(), nullable=False)
    updated_at = db.Column(db.DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    @validates('name')
    def validate_name(self, key, name):
        """Validate role name to be either 'ADMIN' or 'EDITOR'."""
        valid_roles = ['ADMIN', 'EDITOR']
        if name not in valid_roles:
            raise ValueError(f"Role name must be one of {valid_roles}")
        return name

    def __init__(self, name, description=None):
        self.name = name
        self.description = description

    def __repr__(self):
        """String representation of the role."""
        return f"<Role {self.name}>"

    def to_dict(self):
        """Convert role object to dictionary."""
        return {
            'id': self.id,
            'name': self.name,
            'created_at': self.created_at.isoformat(),
            'updated_at': self.updated_at.isoformat()
        }
