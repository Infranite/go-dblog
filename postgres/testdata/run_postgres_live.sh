#!/usr/bin/env bash
set -euo pipefail

image="${1:-postgres:16}"
name="go-dblog-postgres-live-${image//[^a-zA-Z0-9]/-}-$$"
slot="dblog_live_slot"

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
	-p 127.0.0.1::5432 \
	-e POSTGRES_PASSWORD=postgres \
	"$image" \
	-c wal_level=logical \
	-c max_replication_slots=4 \
	-c max_wal_senders=4 >/dev/null

ready=0
for _ in $(seq 1 90); do
	if docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done
if [[ "$ready" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

initialized=0
for _ in $(seq 1 90); do
	if docker logs "$name" 2>&1 | grep -q "PostgreSQL init process complete"; then
		initialized=1
		break
	fi
	sleep 1
done
if [[ "$initialized" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

ready=0
for _ in $(seq 1 90); do
	if docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done
if [[ "$ready" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

port="$(docker port "$name" 5432/tcp | sed 's/.*://')"
if [[ -z "$port" ]]; then
	echo "failed to discover PostgreSQL mapped port" >&2
	exit 1
fi

docker exec "$name" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c \
	"CREATE TABLE public.users (id integer PRIMARY KEY, name text NOT NULL, active boolean NOT NULL);" >/dev/null

docker exec "$name" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c \
	"ALTER TABLE public.users REPLICA IDENTITY FULL;" >/dev/null

docker exec "$name" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c \
	"SELECT pg_create_logical_replication_slot('$slot', 'test_decoding');" >/dev/null

docker exec -i "$name" psql -U postgres -d postgres -v ON_ERROR_STOP=1 <<'SQL' >/dev/null
BEGIN;
INSERT INTO public.users(id, name, active) VALUES (1, 'Ada', true);
UPDATE public.users SET name = 'Ada Lovelace' WHERE id = 1;
DELETE FROM public.users WHERE id = 1;
COMMIT;
SQL

module_dir="$(cd "$(dirname "$0")/.." && pwd)"
export DBLOG_POSTGRES_LIVE_DSN="postgres://postgres:postgres@127.0.0.1:$port/postgres?sslmode=disable"
export DBLOG_POSTGRES_LIVE_SLOT="$slot"

cd "$module_dir"
GOWORK=off go test -race -count=1 -run TestLiveLogicalDecoding .
