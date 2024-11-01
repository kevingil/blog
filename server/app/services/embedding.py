from typing import List, Optional
import numpy as np
from functools import lru_cache
from fastembed import TextEmbedding


class EmbeddingService:
    """Service for generating text embeddings using FastEmbed for pgvector storage."""
    
    def __init__(
        self,
        model_name: str = "BAAI/bge-small-en-v1.5",
        cache_size: int = 1000,
        max_length: int = 512,
    ):
        """
        Initialize the embedding service.
        
        Args:
            model_name: Name of the FastEmbed model to use
            cache_size: Number of embeddings to cache in memory
            max_length: Maximum sequence length for the model
            
        Raises:
            EmbeddingServiceError: If model initialization fails
        """
        try:
            self._model = TextEmbedding(
                model_name=model_name,
                max_length=max_length
            )
            self._get_single_embedding = lru_cache(maxsize=cache_size)(
                self._compute_single_embedding
            )
        except Exception as e:
            raise Exception(f"Failed to initialize embedding model: {str(e)}")

    def _compute_single_embedding(self, text: str) -> np.ndarray:
        """
        Compute embedding for a single text without caching.
        
        Raises:
            EmbeddingServiceError: If embedding computation fails
        """
        try:
            embeddings = list(self._model.embed([text]))
            return embeddings[0]
        except Exception as e:
            raise Exception(f"Failed to compute embedding: {str(e)}")

    def get_embedding(self, text: str, use_cache: bool = True) -> List[float]:
        """
        Get embedding for a single text, formatted for pgvector storage.
        
        Args:
            text: Text to embed
            use_cache: Whether to use cached embeddings
            
        Returns:
            List[float]: Embedding vector ready for pgvector
            
        Raises:
            ValueError: If input text is invalid
            EmbeddingServiceError: If embedding generation fails
        """
        if not isinstance(text, str) or not text.strip():
            raise ValueError("Input must be a non-empty string")

        embedding = (
            self._get_single_embedding(text) if use_cache 
            else self._compute_single_embedding(text)
        )
        
        return embedding.tolist()

    def get_embeddings(
        self, 
        texts: List[str],
        batch_size: Optional[int] = None
    ) -> List[List[float]]:
        """
        Get embeddings for multiple texts, formatted for pgvector storage.
        
        Args:
            texts: List of texts to embed
            batch_size: Optional batch size for processing
            
        Returns:
            List[List[float]]: List of embedding vectors ready for pgvector
            
        Raises:
            ValueError: If input texts are invalid
            EmbeddingServiceError: If embedding generation fails
        """
        if not isinstance(texts, list) or not texts:
            raise ValueError("Input must be a non-empty list of strings")
            
        if not all(isinstance(t, str) and t.strip() for t in texts):
            raise ValueError("All inputs must be non-empty strings")

        try:
            if batch_size:
                embeddings = []
                for i in range(0, len(texts), batch_size):
                    batch = texts[i:i + batch_size]
                    batch_embeddings = list(self._model.embed(batch))
                    embeddings.extend([emb.tolist() for emb in batch_embeddings])
                return embeddings
            
            return [emb.tolist() for emb in self._model.embed(texts)]
        except Exception as e:
            raise Exception(f"Failed to compute embeddings: {str(e)}")
