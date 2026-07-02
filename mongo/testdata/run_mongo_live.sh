#!/usr/bin/env bash
set -euo pipefail

image="${1:-mongo:7.0}"
name="go-dblog-mongo-live-${image//[^a-zA-Z0-9]/-}-$$"

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
	-p 127.0.0.1::27017 \
	"$image" --replSet rs0 --bind_ip_all >/dev/null

ready=0
for _ in $(seq 1 90); do
	if docker exec "$name" mongosh --quiet --eval "db.adminCommand('ping').ok" >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done
if [[ "$ready" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

docker exec "$name" mongosh --quiet --eval \
	'rs.initiate({_id:"rs0", members:[{_id:0, host:"127.0.0.1:27017"}]})' >/dev/null

primary=0
for _ in $(seq 1 90); do
	if [[ "$(docker exec "$name" mongosh --quiet --eval "db.hello().isWritablePrimary")" == "true" ]]; then
		primary=1
		break
	fi
	sleep 1
done
if [[ "$primary" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

port="$(docker port "$name" 27017/tcp | sed 's/.*://')"
if [[ -z "$port" ]]; then
	echo "failed to discover MongoDB mapped port" >&2
	exit 1
fi

module_dir="$(cd "$(dirname "$0")/.." && pwd)"
export DBLOG_MONGO_LIVE_DSN="mongodb://127.0.0.1:$port/?directConnection=true"

cd "$module_dir"
GOWORK=off go test -race -count=1 -run TestLiveChangeStream .
