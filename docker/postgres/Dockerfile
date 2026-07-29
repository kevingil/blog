FROM postgres:17.4-bookworm@sha256:304ab813518754228f9f792f79d6da36359b82d8ecf418096c636725f8c930ad AS pgvector-builder

ARG PGVECTOR_VERSION=0.8.2
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential ca-certificates curl postgresql-server-dev-17 \
    && curl -fsSL "https://github.com/pgvector/pgvector/archive/refs/tags/v${PGVECTOR_VERSION}.tar.gz" -o /tmp/pgvector.tar.gz \
    && mkdir /tmp/pgvector \
    && tar -xzf /tmp/pgvector.tar.gz --strip-components=1 -C /tmp/pgvector \
    && make -C /tmp/pgvector OPTFLAGS="" \
    && make -C /tmp/pgvector install

FROM postgres:17.4-bookworm@sha256:304ab813518754228f9f792f79d6da36359b82d8ecf418096c636725f8c930ad
COPY --from=pgvector-builder /usr/lib/postgresql/17/lib/vector.so /usr/lib/postgresql/17/lib/vector.so
COPY --from=pgvector-builder /usr/share/postgresql/17/extension/vector* /usr/share/postgresql/17/extension/
