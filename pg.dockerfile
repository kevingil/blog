# Dockerfile for PostgreSQL with pgvector
FROM postgres:16.4

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    postgresql-server-dev-16 \
    && rm -rf /var/lib/apt/lists/*

# Clone and install pgvector
RUN git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git \
    && cd pgvector \
    && make \
    && make install

# Add initialization script to enable the extension
RUN echo "CREATE EXTENSION IF NOT EXISTS vector;" > /docker-entrypoint-initdb.d/init-vector.sql
