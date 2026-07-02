#!/usr/bin/env bash
set -euo pipefail

image="${1:-redis:7.2}"
name="go-dblog-redis-live-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	status=$?
	if [[ "$status" -ne 0 ]]; then
		docker logs --tail 160 "$name" >&2 || true
	fi
	docker rm -f "$name" >/dev/null 2>&1 || true
	exit "$status"
}
trap cleanup EXIT

docker pull "$image"
docker run -d --name "$name" \
	-p 127.0.0.1::6379 \
	"$image" \
	redis-server \
	--save "" \
	--appendonly no \
	--repl-diskless-sync no >/dev/null

ready=0
for _ in $(seq 1 90); do
	if docker exec "$name" redis-cli ping >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done
if [[ "$ready" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

port="$(docker port "$name" 6379/tcp | sed 's/.*://')"
if [[ -z "$port" ]]; then
	echo "failed to discover Redis mapped port" >&2
	exit 1
fi

module_dir="$(cd "$(dirname "$0")/.." && pwd)"
export DBLOG_REDIS_LIVE_ADDR="127.0.0.1:$port"
export DBLOG_REDIS_LIVE_DSN="redis://127.0.0.1:$port/0"

cd "$module_dir"
GOWORK=off go test -race -count=1 -run TestLiveReplicationStream .
