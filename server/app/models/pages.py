from sqlalchemy import Column, Integer, String, Text, DateTime, JSON
from sqlalchemy.sql import func
from app.extensions import db

class AboutPage(db.Model):
    """About Page model for storing information about the q'About' section."""
    __tablename__ = 'about_page'

    id = Column(Integer, primary_key=True, autoincrement=True)
    title = Column(String(255), nullable=False)
    content = Column(Text, nullable=False)
    profile_image = Column(String(255), nullable=True)
    meta_description = Column(String(255), nullable=True)
    last_updated = Column(DateTime, nullable=False, default=func.now())

    def __repr__(self):
        """String representation of the About Page."""
        return f"<AboutPage {self.title}>"

    def to_dict(self):
        """Convert about page object to dictionary."""
        return {
            'id': self.id,
            'title': self.title,
            'content': self.content,
            'profile_image': self.profile_image,
            'meta_description': self.meta_description,
            'last_updated': self.last_updated.isoformat()
        }


class ContactPage(db.Model):
    """Contact Page model for storing information about the 'Contact' section."""
    __tablename__ = 'contact_page'

    id = Column(Integer, primary_key=True, autoincrement=True)
    title = Column(String(255), nullable=False)
    content = Column(Text, nullable=False)
    email_address = Column(String(255), nullable=False)
    social_links = Column(JSON, nullable=True)
    meta_description = Column(String(255), nullable=True)
    last_updated = Column(DateTime, nullable=False, default=func.now())

    def __repr__(self):
        """String representation of the Contact Page."""
        return f"<ContactPage {self.title}>"

    def to_dict(self):
        """Convert contact page object to dictionary."""
        return {
            'id': self.id,
            'title': self.title,
            'content': self.content,
            'email_address': self.email_address,
            'social_links': self.social_links,
            'meta_description': self.meta_description,
            'last_updated': self.last_updated.isoformat()
        }
