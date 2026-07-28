#!/usr/bin/env sh
set -eu

mode="${1:-blocking}"
repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$repo_root/backend"

case "$mode" in
  blocking)
    cargo test --locked --lib -- --test-threads=1
    for target in \
      article_http_database \
      article_repository \
      article_service \
      auth \
      auth_database \
      chat_repository \
      chat_service \
      content_crud_http_database \
      copilot_contract \
      copilot_manager \
      data_domain_http \
      datasource_repository \
      datasource_service \
      exa_client \
      fetch_extract \
      health \
      image_http \
      image_service \
      ml_agent \
      ml_research_tools \
      ml_services \
      ml_tools \
      openai_client \
      organization_repository \
      organization_service \
      page_repository \
      page_service \
      profile_repository \
      profile_service \
      project_repository \
      project_service \
      s3_object_store \
      source_service \
      storage_repository \
      storage_service \
      support_http \
      tag_repository \
      tag_service \
      taskrun_repository \
      taskrun_service \
      websocket_contract \
      websocket_network \
      worker_core \
      worker_domains \
      worker_http \
      worker_adapters
    do
      cargo test --locked --test "$target" -- --test-threads=1
    done
    ;;
  insights)
    cargo test --locked --test insight_exceptions -- --test-threads=1
    ;;
  *)
    echo "usage: $0 [blocking|insights]" >&2
    exit 2
    ;;
esac
