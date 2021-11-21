#!/bin/bash
TST=0
function assert() {
  TST=$((TST+1))
  EP=$1
  EXPECTED=$2
  RESULT=$(curl --cacert pgroute66.crt "https://localhost:8443/v1/${EP}" 2>/dev/null | xargs)
  if [ "${RESULT}" = "${EXPECTED}" ]; then
    echo "test${TST}: OK"
  else
    echo "test${TST}: EROR: expected '${EXPECTED}', but got '${RESULT}'"
    return 1
  fi
}

#set -x
set -e

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout pgroute66.key -out pgroute66.crt -subj "/C=NL/ST=Zuid Holland/L=Nederland/O=Mannem Solutions/CN=localhost"
CERT=$(cat pgroute66.crt | base64)
KEY=$(cat pgroute66.key | base64)
sed -i "s/b64cert:.*/b64cert: ${CERT}/;s/b64key:.*/b64key: ${KEY}/" config.yaml

docker-compose down && docker rmi pgroute66_postgres pgroute66_pgroute66  || echo new install
docker-compose up -d --scale postgres=3
for ((i=1;i<4;i++)); do
  docker exec -ti pgroute66_postgres_$i /entrypoint.sh background
done

docker-compose up -d pgroute66
assert primary 'host1'
assert primaries '[ host1 ]'
assert standbys '[ host2, host3 ]'

docker exec -ti pgroute66_postgres_2 /entrypoint.sh promote
assert primary ''
assert primaries '[ host1, host2 ]'
assert standbys '[ host3 ]'

docker exec -ti pgroute66_postgres_1 /entrypoint.sh rebuild
assert primary 'host2'
assert primaries '[ host2 ]'
assert standbys '[ host1, host3 ]'

echo "All is as expected"
