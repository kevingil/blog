from pydantic import BaseModel, Field
from typing import List, Optional
from datetime import datetime
from apispec_pydantic_plugin import Registry


class TagSchema(BaseModel):
    """Base schema for Tag."""
    name: str = Field(
        ...,
        min_length=1,
        max_length=255,
        description="The name of the tag"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "name": "technology"
            }
        }


class TagCreate(TagSchema):
    """Schema for creating a new tag."""
    pass


class TagResponse(TagSchema):
    """Schema for tag response including ID."""
    id: int = Field(..., description="The unique identifier of the tag")

    class Config:
        from_attributes = True
        json_schema_extra = {
            "example": {
                "id": 1,
                "name": "technology"
            }
        }


class ArticleSchema(BaseModel):
    """Base schema for Article."""
    title: str = Field(
        ...,
        min_length=1,
        max_length=255,
        description="The title of the article"
    )
    content: str = Field(
        ...,
        min_length=1,
        description="The main content of the article"
    )
    slug: str = Field(
        ...,
        min_length=1,
        max_length=255,
        description="URL-friendly version of the title"
    )
    image: Optional[str] = Field(
        None,
        description="URL of the article's featured image"
    )
    is_draft: bool = Field(
        True,
        description="Whether the article is in draft status"
    )
    tags: List[str] = Field(
        default_factory=list,
        description="List of tag names associated with the article"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "title": "Getting Started with Python",
                "content": "Python is a versatile programming language...",
                "slug": "getting-started-with-python",
                "image": "https://example.com/images/python.jpg",
                "is_draft": True,
                "tags": ["python", "programming", "tutorial"]
            }
        }


class ArticleCreate(ArticleSchema):
    """Schema for creating a new article."""
    pass


class ArticleUpdate(BaseModel):
    """Schema for updating an existing article."""
    title: Optional[str] = Field(
        None,
        min_length=1,
        max_length=255,
        description="The title of the article"
    )
    content: Optional[str] = Field(
        None,
        min_length=1,
        description="The main content of the article"
    )
    image: Optional[str] = Field(
        None,
        description="URL of the article's featured image"
    )
    is_draft: Optional[bool] = Field(
        None,
        description="Whether the article is in draft status"
    )
    tags: Optional[List[str]] = Field(
        None,
        description="List of tag names associated with the article"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "title": "Updated: Getting Started with Python",
                "is_draft": False,
                "tags": ["python", "beginner"]
            }
        }


class ArticleResponse(ArticleSchema):
    """Schema for article response including all fields."""
    id: int = Field(..., description="The unique identifier of the article")
    author_id: int = Field(..., description="The ID of the article's author")
    created_at: datetime = Field(..., description="When the article was created")
    updated_at: datetime = Field(..., description="When the article was last updated")
    embedding: Optional[List[float]] = Field(
        None,
        description="Vector embedding of the article content"
    )
    tags: List[TagResponse] = Field(
        default_factory=list,
        description="List of tags associated with the article"
    )

    class Config:
        from_attributes = True
        json_schema_extra = {
            "example": {
                "id": 1,
                "title": "Getting Started with Python",
                "content": "Python is a versatile programming language...",
                "slug": "getting-started-with-python",
                "image": "https://example.com/images/python.jpg",
                "is_draft": False,
                "author_id": 1,
                "created_at": "2024-10-31T12:00:00Z",
                "updated_at": "2024-10-31T12:00:00Z",
                "embedding": [0.1, 0.2, 0.3],
                "tags": [
                    {"id": 1, "name": "python"},
                    {"id": 2, "name": "programming"}
                ]
            }
        }


# APISpec
Registry.register(TagSchema)
Registry.register(TagCreate)
Registry.register(TagResponse)
Registry.register(ArticleSchema)
Registry.register(ArticleCreate)
Registry.register(ArticleUpdate)
Registry.register(ArticleResponse)
