#!/usr/bin/env bash
set -euo pipefail

image="${1:-mongo:7.0}"
out="${2:-$(dirname "$0")/oplog.jsonl}"
name="go-dblog-mongo-${image//[^a-zA-Z0-9]/-}-$$"

cleanup() {
	status=$?
	if [[ "$status" -ne 0 ]]; then
		if [[ -n "${out:-}" && -f "$out" ]]; then
			echo "generated MongoDB oplog output:" >&2
			sed -n '1,120p' "$out" >&2
		fi
		docker logs --tail 160 "$name" >&2 || true
	fi
	docker rm -f "$name" >/dev/null 2>&1 || true
	exit "$status"
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

docker exec "$name" mongosh --quiet --norc --eval '
const database = db.getSiblingDB("dblog_ci");
database.users.drop();
database.users.insertOne({_id: 1, name: "Ada", active: true});
database.users.updateOne({_id: 1}, {$set: {name: "Ada Lovelace"}});
database.users.deleteOne({_id: 1});
' >/dev/null

mkdir -p "$(dirname "$out")"
docker exec "$name" mongosh --quiet --norc --eval '
let seen = 0;
db.getSiblingDB("local").getCollection("oplog.rs")
  .find({ns: "dblog_ci.users", op: {$in: ["i", "u", "d"]}})
  .sort({$natural: 1})
  .forEach((doc) => {
    print(EJSON.stringify(doc, null, 0, {relaxed: true}));
    seen++;
  });
if (seen < 3) {
  quit(2);
}
' >"$out"

grep -q '"op":"i"' "$out"
grep -q '"op":"u"' "$out"
grep -q '"op":"d"' "$out"
ls -lh "$out"
