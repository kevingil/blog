from app.extensions import celery, embeddings, db
from app.models import Article

@celery.task(
    bind=True,
    max_retries=3,
    default_retry_delay=300,  
    name='tasks.generate_article_embedding'
)
def generate_article_embedding(self, article_id: int):
    """Generate and save embedding for an article.
    
    Args:
        article_id: ID of the article to generate embedding for
    """
    try:
        article = Article.query.get(article_id)
        if not article:
            celery.logger.warning(f"Article {article_id} not found")
            return
        
        # Combine relevant text fields for embedding
        text_to_embed = f"{article.title}\n{article.description}\n{article.content}"
        
        # Generate embedding
        embedding_vector = embeddings.get_embedding(text_to_embed)
        
        # Update article with embedding
        article.embedding = embedding_vector
        db.session.commit()
        
        celery.logger.info(f"Successfully generated embedding for article {article_id}")
        
    except Exception as e:
        celery.logger.error(f"Error generating embedding for article {article_id}: {str(e)}")
        self.retry(exc=e)
