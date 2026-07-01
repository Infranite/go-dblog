#!/usr/bin/env bash
set -euo pipefail

image="${1:-mongo:7.0}"
out="${2:-$(dirname "$0")/oplog.jsonl}"
name="go-dblog-mongo-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker pull "$image"
docker run -d --name "$name" "$image" --replSet rs0 --bind_ip_all >/dev/null

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

docker exec "$name" mongosh --quiet --eval "rs.initiate()" >/dev/null

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

docker exec "$name" mongosh --quiet <<'JS' >/dev/null
const db = db.getSiblingDB("dblog_ci");
db.users.drop();
JS

mkdir -p "$(dirname "$out")"
docker exec "$name" mongosh --quiet <<'JS' >"$out"
const db = db.getSiblingDB("dblog_ci");
const stream = db.users.watch([], {fullDocument: "updateLookup"});
db.users.insertOne({_id: 1, name: "Ada", active: true});
db.users.updateOne({_id: 1}, {$set: {name: "Ada Lovelace"}});
db.users.deleteOne({_id: 1});

let seen = 0;
const deadline = Date.now() + 10000;
while (seen < 3 && Date.now() < deadline) {
  const change = stream.tryNext();
  if (change) {
    print(EJSON.stringify(change));
    seen++;
    continue;
  }
  sleep(100);
}
stream.close();
if (seen !== 3) {
  quit(2);
}
JS

grep -q '"operationType":"insert"' "$out"
grep -q '"operationType":"update"' "$out"
grep -q '"operationType":"delete"' "$out"
ls -lh "$out"
