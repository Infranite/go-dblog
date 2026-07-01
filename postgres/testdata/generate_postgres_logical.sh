#!/usr/bin/env bash
set -euo pipefail

image="${1:-postgres:16}"
out="${2:-$(dirname "$0")/test_decoding.log}"
name="go-dblog-postgres-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker pull "$image"
docker run -d --name "$name" \
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

docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 <<'SQL' >/dev/null
CREATE DATABASE dblog_ci;
SQL

docker exec "$name" psql -U postgres -d dblog_ci -v ON_ERROR_STOP=1 <<'SQL' >/dev/null
CREATE TABLE public.users (
	id integer PRIMARY KEY,
	name text NOT NULL,
	active boolean NOT NULL
);
SELECT pg_create_logical_replication_slot('dblog_ci_slot', 'test_decoding');
BEGIN;
INSERT INTO public.users(id, name, active) VALUES (1, 'Ada', true);
UPDATE public.users SET name = 'Ada Lovelace' WHERE id = 1;
DELETE FROM public.users WHERE id = 1;
COMMIT;
SQL

mkdir -p "$(dirname "$out")"
docker exec "$name" psql -U postgres -d dblog_ci -At \
	-c "SELECT data FROM pg_logical_slot_get_changes('dblog_ci_slot', NULL, NULL);" >"$out"

grep -q '^BEGIN ' "$out"
grep -q 'table public.users: INSERT:' "$out"
grep -q 'table public.users: UPDATE:' "$out"
grep -q 'table public.users: DELETE:' "$out"
grep -q '^COMMIT ' "$out"
ls -lh "$out"
