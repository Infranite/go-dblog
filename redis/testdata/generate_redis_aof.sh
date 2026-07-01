#!/usr/bin/env bash
set -euo pipefail

image="${1:-redis:7.2}"
out="${2:-$(dirname "$0")/appendonly.aof}"
name="go-dblog-redis-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker pull "$image"
docker run -d --name "$name" "$image" \
	redis-server \
	--appendonly yes \
	--appendfsync always \
	--aof-use-rdb-preamble no \
	--save "" >/dev/null

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

docker exec "$name" redis-cli HSET user:1 name Ada >/dev/null
docker exec "$name" redis-cli SADD tags go >/dev/null
docker exec "$name" redis-cli LPUSH queue job-1 >/dev/null
docker exec "$name" redis-cli INCR counter >/dev/null

aof_path="$(docker exec "$name" sh -c "find /data -type f \\( -name '*incr.aof' -o -name 'appendonly.aof' \\) | sort | tail -n 1")"
if [[ -z "$aof_path" ]]; then
	docker logs "$name"
	echo "failed to discover Redis AOF path" >&2
	exit 1
fi

mkdir -p "$(dirname "$out")"
docker cp "$name:$aof_path" "$out"
grep -aq 'HSET' "$out"
grep -aq 'SADD' "$out"
grep -aq 'LPUSH' "$out"
grep -aq 'INCR' "$out"
ls -lh "$out"
