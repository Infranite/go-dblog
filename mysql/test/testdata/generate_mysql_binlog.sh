#!/usr/bin/env bash
set -euo pipefail

image="${1:-mysql:8.0}"
out="${2:-$(dirname "$0")/mysql-bin.000004}"
name="go-dblog-mysql-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker pull "$image"
docker run -d --name "$name" \
	-e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
	"$image" \
	--server-id=1 \
	--log-bin=/var/lib/mysql/mysql-bin \
	--binlog-format=ROW \
	--binlog-row-image=FULL >/dev/null

ready=0
for _ in $(seq 1 90); do
	if docker exec "$name" mysqladmin ping -uroot --silent >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 1
done
if [[ "$ready" != 1 ]]; then
	docker logs "$name"
	exit 1
fi

binlog_file="$(docker exec "$name" mysql -N -uroot -e "SHOW BINARY LOG STATUS" 2>/dev/null | awk 'NR == 1 {print $1}' || true)"
if [[ -z "$binlog_file" ]]; then
	binlog_file="$(docker exec "$name" mysql -N -uroot -e "SHOW MASTER STATUS" 2>/dev/null | awk 'NR == 1 {print $1}' || true)"
fi
if [[ -z "$binlog_file" ]]; then
	docker logs "$name"
	echo "failed to discover active MySQL binlog" >&2
	exit 1
fi

docker exec "$name" mysql -uroot <<'SQL'
CREATE DATABASE dblog_ci;
USE dblog_ci;
SET SESSION binlog_format = 'ROW';
CREATE TABLE events (
	id BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(64) NOT NULL,
	amount DECIMAL(10,2) NOT NULL,
	note VARBINARY(64) NULL,
	created_at DATETIME NOT NULL
) ENGINE=InnoDB;
INSERT INTO events(name, amount, note, created_at)
VALUES ('alpha', 12.30, X'010203', '2026-07-01 00:00:00');
UPDATE events SET amount = 13.40 WHERE name = 'alpha';
DELETE FROM events WHERE name = 'alpha';
FLUSH LOGS;
SQL

mkdir -p "$(dirname "$out")"
docker cp "$name:/var/lib/mysql/$binlog_file" "$out"
ls -lh "$out"
