from sqlalchemy import Column, Integer, String, Text, Boolean, ForeignKey, DateTime, Float, ARRAY, UniqueConstraint
from sqlalchemy.sql import func
from blog.extensions import db

class Article(db.Model):
    """Articles model for storing articles with vector embeddings."""
    __tablename__ = 'articles'

    id = Column(Integer, primary_key=True, autoincrement=True)
    image = Column(Text, nullable=True)
    slug = Column(String(255), unique=True, nullable=False)
    title = Column(String(255), nullable=False)
    content = Column(Text, nullable=False)
    author = Column(Integer, ForeignKey('users.id'), nullable=False)
    created_at = Column(DateTime, nullable=False, default=func.now()) 
    is_draft = Column(Boolean, nullable=False, default=True)
    embedding = Column(ARRAY(Float), nullable=True) 

    def __repr__(self):
        return f"<Article {self.title}>"

    def to_dict(self):
        return {
            'id': self.id,
            'image': self.image,
            'slug': self.slug,
            'title': self.title,
            'content': self.content,
            'author': self.author,
            'created_at': self.created_at.isoformat(),
            'is_draft': self.is_draft,
            'embedding': self.embedding
        }


class Tag(db.Model):
    """Tags model for storing unique article tags."""
    __tablename__ = 'tags'

    id = Column(Integer, primary_key=True, autoincrement=True)
    name = Column(String(255), unique=True, nullable=True)

    def __repr__(self):
        return f"<Tag {self.name}>"

    def to_dict(self):
        return {
            'id': self.id,
            'name': self.name
        }
        
        
class ArticleTag(db.Model):
    """ArticleTags junction model to link articles and tags."""
    __tablename__ = 'article_tags'

    article_id = Column(Integer, ForeignKey('articles.id'), nullable=False)
    tag_id = Column(Integer, ForeignKey('tags.id'), nullable=False)

    __table_args__ = (UniqueConstraint('article_id', 'tag_id', name='uix_article_tag'),)

    def __repr__(self):
        return f"<ArticleTag article_id={self.article_id}, tag_id={self.tag_id}>"

    def to_dict(self):
        return {
            'article_id': self.article_id,
            'tag_id': self.tag_id
        }
