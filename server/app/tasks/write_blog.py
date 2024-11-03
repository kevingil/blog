from datetime import datetime
from typing import Dict, Any
from slugify import slugify
from sqlalchemy.exc import SQLAlchemyError
from app.services.agents.writting_agent import BlogState, BlogWriterAgent
from app.models.article import Article, Tag, ArticleTag
from app.extensions import db, celery

@celery.task(name="tasks.write_blog")
def write_blog(topic: str, author_id: int, tags: list[str] = None) -> Dict[str, Any]:
    """
    Celery task to write a blog post and save it to the database.
    
    Args:
        topic (str): The blog topic to write about, from user prompt
        author_id (int): ID of the author (user) creating the blog
        tags (list[str], optional): List of tags to associate with the article
        
    Returns:
        Dict[str, Any]: Status and article information
    """
    try:
        # Initialize state and agent
        state = BlogState()
        state.topic = topic
        agent = BlogWriterAgent()
        
        # Create and run workflow
        workflow = agent.create_workflow()
        result = workflow.run(state.dict())
        
        # Create slug from topic
        slug = slugify(topic)
        
        # Create new article
        article = Article(
            title=topic,
            slug=slug,
            content=result["draft"],
            author_id=author_id,
            is_draft=True  # Set as draft initially
        )
        
        # Add article to database
        db.session.add(article)
        
        # Handle tags if provided
        if tags:
            for tag_name in tags:
                # Get or create tag
                tag = Tag.query.filter_by(name=tag_name).first()
                if not tag:
                    tag = Tag(name=tag_name)
                    db.session.add(tag)
                    db.session.flush()  # Get tag ID
                
                # Create article-tag association
                article_tag = ArticleTag(
                    article_id=article.id,
                    tag_id=tag.id
                )
                db.session.add(article_tag)
        
        # Commit changes
        db.session.commit()
        
        celery.logger.info(f"Successfully created article: {article.id} - {article.title}")
        
        return {
            "status": "success",
            "article": {
                "id": article.id,
                "slug": article.slug,
                "title": article.title,
                "created_at": article.created_at.isoformat(),
                "is_draft": article.is_draft
            },
            "outline": result["outline"],
            "research_data": result["research_data"]
        }
        
    except SQLAlchemyError as e:
        celery.logger.error(f"Database error while creating article: {str(e)}")
        db.session.rollback()
        return {
            "status": "error",
            "error": "Database error occurred",
            "details": str(e)
        }
        
    except Exception as e:
        celery.logger.error(f"Error while writing blog: {str(e)}")
        db.session.rollback()
        return {
            "status": "error",
            "error": "An unexpected error occurred",
            "details": str(e)
        }
