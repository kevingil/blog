from sqlalchemy.sql import func
from blog.extensions import db

class Project(db.Model):
    """Project model for storing project information."""
    __tablename__ = 'projects'

    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    title = db.Column(db.String(255), nullable=False)
    description = db.Column(db.Text, nullable=False)
    url = db.Column(db.String(255), nullable=False)
    image = db.Column(db.String(255), nullable=True)
    created_at = db.Column(db.DateTime, default=func.now(), nullable=False)
    updated_at = db.Column(db.DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    def __repr__(self):
        """String representation of the project."""
        return f"<Project {self.title}>"

    def to_dict(self):
        """Convert project object to dictionary."""
        return {
            'id': self.id,
            'title': self.title,
            'description': self.description,
            'url': self.url,
            'image': self.image,
            'created_at': self.created_at.isoformat(),
            'updated_at': self.updated_at.isoformat()
        }
