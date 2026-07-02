#!/usr/bin/env bash
set -euo pipefail

image="${1:-mysql:8.4}"
name="go-dblog-mysql-live-${image//[^a-zA-Z0-9]/-}-$$"

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
	-e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
	-e MYSQL_ROOT_HOST=% \
	-p 127.0.0.1::3306 \
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

reset_binary_logs() {
	if docker exec "$name" mysql -uroot -e "RESET MASTER" >/dev/null 2>&1; then
		return
	fi
	docker exec "$name" mysql -uroot -e "RESET BINARY LOGS AND GTIDS" >/dev/null
}

docker exec -i "$name" mysql -uroot <<'SQL'
CREATE USER IF NOT EXISTS 'dblog'@'%' IDENTIFIED BY 'dblog';
GRANT ALL PRIVILEGES ON *.* TO 'dblog'@'%';
FLUSH PRIVILEGES;
SQL
reset_binary_logs

port="$(docker port "$name" 3306/tcp | sed 's/.*://')"
if [[ -z "$port" ]]; then
	echo "failed to discover MySQL mapped port" >&2
	exit 1
fi

module_dir="$(cd "$(dirname "$0")/../.." && pwd)"
export DBLOG_MYSQL_LIVE_ADDR="127.0.0.1:$port"
export DBLOG_MYSQL_LIVE_DSN="mysql://dblog:dblog@127.0.0.1:$port/?server_id=1002"
export DBLOG_MYSQL_LIVE_USER="dblog"
export DBLOG_MYSQL_LIVE_PASSWORD="dblog"

cd "$module_dir"
GOWORK=off go test -race -count=1 -run TestLiveReplicationStream .
