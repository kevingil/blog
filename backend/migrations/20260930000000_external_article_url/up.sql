ALTER TABLE article ADD COLUMN external_url TEXT;

CREATE UNIQUE INDEX article_external_url_key
    ON article (external_url)
    WHERE external_url IS NOT NULL;
