from sqlalchemy import Column, Integer, String, Text, DateTime, JSON, Boolean, Enum, ForeignKey
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship
from app.extensions import db

class Page(db.Model):
    """Base model for all pages in the system."""
    __tablename__ = 'pages'

    id = Column(Integer, primary_key=True, autoincrement=True)
    slug = Column(String(100), unique=True, nullable=False)
    title = Column(String(255), nullable=False)
    meta_description = Column(String(255), nullable=True)
    is_active = Column(Boolean, default=True)
    is_system_page = Column(Boolean, default=False)
    show_in_nav = Column(Boolean, default=False)
    nav_order = Column(Integer, nullable=True)
    created_at = Column(DateTime, nullable=False, default=func.now())
    updated_at = Column(DateTime, nullable=False, default=func.now(), onupdate=func.now())

    # Relationship to the content versions
    contents = relationship("PageContent", back_populates="page", order_by="desc(PageContent.version)")

    def __repr__(self):
        """String representation of the Page."""
        return f"<Page {self.title} ({self.page_type})>"

    @property
    def current_content(self):
        """Get the current content version."""
        return next((c for c in self.contents if c.is_current), None)

    def to_dict(self):
        """Convert page object to dictionary."""
        current = self.current_content
        
        result = {
            'id': self.id,
            'slug': self.slug,
            'title': self.title,
            'meta_description': self.meta_description,
            'is_active': self.is_active,
            'show_in_nav': self.show_in_nav,
            'nav_order': self.nav_order,
            'updated_at': self.updated_at.isoformat()
        }

        if current:
            result.update(current.to_dict())

        return result

class PageContent(db.Model):
    """Content versions for pages with metadata."""
    __tablename__ = 'page_contents'

    id = Column(Integer, primary_key=True, autoincrement=True)
    page_id = Column(Integer, ForeignKey('pages.id', ondelete='CASCADE'), nullable=False)
    content = Column(Text, nullable=False)
    content_type = Column(Enum('html', 'markdown', 'plain_text', name='content_type'), default='html')
    created_at = Column(DateTime, nullable=False, default=func.now())
    updated_at = Column(DateTime, nullable=True, default=func.now())

    # Relationship to the parent page
    page = relationship("Page", back_populates="contents")

    def to_dict(self):
        """Convert content to dictionary, excluding None values."""
        return {
            k: v for k, v in {
                'content': self.content,
                'content_type': self.content_type,
                'created_at': self.created_at.isoformat(),
                'updated_at': self.updated_at.isoformat()
            }.items() if v is not None
        }
