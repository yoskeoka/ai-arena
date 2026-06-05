#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
data_dir=${SEAWEED_DATA_DIR:-"$repo_root/.local/seaweed"}
bucket=${SEAWEED_BUCKET:-ai-arena-local}
endpoint=${SEAWEED_ENDPOINT:-http://127.0.0.1:8333}
aws_cli_image=${AWS_CLI_IMAGE:-amazon/aws-cli:2}

case "$data_dir" in
  ""|"/")
    echo "refusing to reset unsafe SEAWEED_DATA_DIR: $data_dir" >&2
    exit 1
    ;;
esac

SEAWEED_DATA_DIR="$data_dir" docker compose -f "$repo_root/tools/dev/seaweed-compose.yml" down -v >/dev/null 2>&1 || true
rm -rf "$data_dir"
mkdir -p "$data_dir"

SEAWEED_DATA_DIR="$data_dir" docker compose -f "$repo_root/tools/dev/seaweed-compose.yml" up -d seaweed

attempt=0
until curl -sS -o /dev/null "$endpoint/"; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "SeaweedFS did not become ready at $endpoint" >&2
    exit 1
  fi
  sleep 1
done

docker run --rm --network host \
  -e AWS_ACCESS_KEY_ID=admin \
  -e AWS_SECRET_ACCESS_KEY=secret \
  "$aws_cli_image" \
  s3api create-bucket \
  --bucket "$bucket" \
  --endpoint-url "$endpoint" \
  --region us-east-1 >/dev/null

if [ "${SEAWEED_SEED_FILE:-}" != "" ] && [ "${SEAWEED_SEED_KEY:-}" != "" ]; then
  docker run --rm --network host \
    -e AWS_ACCESS_KEY_ID=admin \
    -e AWS_SECRET_ACCESS_KEY=secret \
    -v "$repo_root:/work" \
    "$aws_cli_image" \
    s3 cp "/work/${SEAWEED_SEED_FILE}" "s3://$bucket/${SEAWEED_SEED_KEY}" \
    --endpoint-url "$endpoint" \
    --region us-east-1 >/dev/null
fi

echo "SeaweedFS ready: endpoint=$endpoint bucket=$bucket data_dir=$data_dir"
