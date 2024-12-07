from sqlalchemy.ext.hybrid import hybrid_property
from sqlalchemy.orm import validates
from sqlalchemy.sql import func
from app.models.article import Article
from app.extensions import db, pwd_context
import re


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


class User(db.Model):
    """User model for storing user account information."""
    __tablename__ = 'users'

    # Primary fields
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(80), unique=True, nullable=False, index=True)
    email = db.Column(db.String(80), unique=True, nullable=False, index=True)
    _password = db.Column("password", db.String(255), nullable=False)
    avatar = db.Column(db.String(80), nullable=True)
    
    # Relationships
    role_id = db.Column(db.Integer, db.ForeignKey('roles.id'))
    role = db.relationship('Role', backref=db.backref('users', lazy='dynamic'))
    posts = db.relationship('Article', backref='author', lazy='dynamic')
    
    # Timestamps
    created_at = db.Column(db.DateTime, default=func.now(), nullable=False)
    updated_at = db.Column(db.DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    @hybrid_property
    def password(self):
        """Prevent password from being accessed."""
        raise AttributeError('Password is not a readable attribute')

    @password.setter
    def password(self, value):
        """Hash password on set."""
        self._password = pwd_context.hash(value)

    @validates('email')
    def validate_email(self, key, email):
        """Validate email format."""
        if not email:
            raise ValueError('Email is required')
        if not re.match(r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$', email):
            raise ValueError('Invalid email format')
        return email.lower()

    def verify_password(self, password):
        """Check if provided password matches stored hash."""
        return pwd_context.verify(password, self._password)

    def __repr__(self):
        """String representation of the user."""
        return f"<User {self.email}>"

    def to_dict(self):
        """Convert user object to dictionary."""
        return {
            'id': self.id,
            'name': self.name,
            'email': self.email,
            'avatar': self.avatar,
            'role_id': self.role_id,
            'created_at': self.created_at.isoformat(),
            'updated_at': self.updated_at.isoformat()
        }
