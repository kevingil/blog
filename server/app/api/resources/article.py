from flask import request
from flask_restful import Resource
from flask_jwt_extended import jwt_required
from sqlalchemy.orm import joinedload
from app.extensions import db
from app.commons.pagination import paginate
from app.api.schemas import ArticleSchema
from app.models import Article, Tag, ArticleTag


class ArticleResource(Resource):
    """Single object resource
    
    ---
    get:
      tags:
        - api
      summary: Get an article
      description: Get a single article by ID
      parameters:
        - in: path
          name: article_id
          schema:
            type: integer
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  article: ArticleSchema
        404:
          description: article does not exists
    put:
      tags:
        - api
      summary: Update an article
      description: Update a single article by ID
      parameters:
        - in: path
          name: article_id
          schema:
            type: integer
      requestBody:
        content:
          application/json:
            schema:
              ArticleSchema
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  msg:
                    type: string
                    example: article updated
                  article: ArticleSchema
        404:
          description: article does not exists
    delete:
      tags:
        - api
      summary: Delete an article
      description: Delete a single article by ID
      parameters:
        - in: path
          name: article_id
          schema:
            type: integer
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  msg:
                    type: string
                    example: article deleted
        404:
          description: article does not exists
    """

    method_decorators = [jwt_required()]

    def get(self, article_id):
        schema = ArticleSchema()
        article = Article.query.options(
            joinedload(Article.tags)
        ).get_or_404(article_id)
        return {"article": schema.dump(article)}

    def put(self, article_id):
        schema = ArticleSchema(partial=True)
        article = Article.query.get_or_404(article_id)
        article = schema.load(request.json, instance=article)

        if 'tags' in request.json:
            # Remove existing tags
            ArticleTag.query.filter_by(article_id=article.id).delete()
            
            # Add new tags
            for tag_name in request.json['tags']:
                tag = Tag.query.filter_by(name=tag_name).first()
                if not tag:
                    tag = Tag(name=tag_name)
                    db.session.add(tag)
                    db.session.flush()
                article_tag = ArticleTag(article_id=article.id, tag_id=tag.id)
                db.session.add(article_tag)

        db.session.commit()

        return {"msg": "article updated", "article": schema.dump(article)}

    def delete(self, article_id):
        article = Article.query.get_or_404(article_id)
        db.session.delete(article)
        db.session.commit()

        return {"msg": "article deleted"}


class ArticleList(Resource):
    """Creation and get_all
    
    ---
    get:
      tags:
        - api
      summary: Get a list of articles
      description: Get a list of paginated articles
      responses:
        200:
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/PaginatedResult'
                  - type: object
                    properties:
                      results:
                        type: array
                        items:
                          $ref: '#/components/schemas/ArticleSchema'
    post:
      tags:
        - api
      summary: Create an article
      description: Create a new article
      requestBody:
        content:
          application/json:
            schema:
              ArticleSchema
      responses:
        201:
          content:
            application/json:
              schema:
                type: object
                properties:
                  msg:
                    type: string
                    example: article created
                  article: ArticleSchema
    """

    method_decorators = [jwt_required()]

    def get(self):
        schema = ArticleSchema(many=True)
        query = Article.query.options(joinedload(Article.tags))
        return paginate(query, schema)

    def post(self):
        schema = ArticleSchema()
        article = schema.load(request.json)
        article.author_id = request.current_user.id

        if 'tags' in request.json:
            for tag_name in request.json['tags']:
                tag = Tag.query.filter_by(name=tag_name).first()
                if not tag:
                    tag = Tag(name=tag_name)
                    db.session.add(tag)
                    db.session.flush()
                article_tag = ArticleTag(article_id=article.id, tag_id=tag.id)
                db.session.add(article_tag)

        db.session.add(article)
        db.session.commit()

        return {"msg": "article created", "article": schema.dump(article)}, 201
