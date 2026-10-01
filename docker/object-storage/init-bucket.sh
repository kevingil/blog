#!/bin/sh
# Create the local S3 bucket and grant anonymous GetObject so the browser
# can load objects from S3_URL_PREFIX (http://localhost:9000/blog/...).
set -eu

endpoint="${S3_ENDPOINT:-http://object-storage:9000}"
bucket="${S3_BUCKET:-blog}"

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-blog}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-blog-local-secret}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
export AWS_EC2_METADATA_DISABLED=true

echo "Waiting for S3 at ${endpoint}..."
ready=0
i=0
while [ "$i" -lt 60 ]; do
  if aws --endpoint-url "$endpoint" s3api list-buckets >/dev/null 2>&1; then
    ready=1
    break
  fi
  i=$((i + 1))
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "S3 endpoint ${endpoint} did not become ready" >&2
  exit 1
fi

if ! aws --endpoint-url "$endpoint" s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
  aws --endpoint-url "$endpoint" s3api create-bucket --bucket "$bucket"
fi

aws --endpoint-url "$endpoint" s3api put-bucket-policy --bucket "$bucket" --policy "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":[\"*\"]},\"Action\":[\"s3:GetObject\"],\"Resource\":[\"arn:aws:s3:::${bucket}/*\"]}]}"

echo "Bucket ${bucket} is ready for public download."
