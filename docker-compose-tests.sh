#!/bin/bash
TST=0
function assert() {
  TST=$((TST+1))
  EP=$1
  EXPECTED=$2
  if [ -e pgroute66.crt ]; then
    RESULT=$(curl --cacert pgroute66.crt "https://localhost:8443/v1/${EP}" | xargs)
  else
    RESULT=$(curl "http://localhost:8080/v1/${EP}" | xargs)
  fi
  if [ "${RESULT}" = "${EXPECTED}" ]; then
    echo "test${TST}: OK"
  else
    echo "test${TST}: EROR: expected '${EXPECTED}', but got '${RESULT}'"
    docker-compose logs pgroute66 postgres
    return 1
  fi
}

#set -x
set -e

if [ ! -f ./test/config.yaml ]; then
  mkdir -p ./test
  cp config.yaml ./test/config.yaml
  echo "generating openssl cert"
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout pgroute66.key -out pgroute66.crt -subj "/C=NL/ST=Zuid Holland/L=Nederland/O=Mannem Solutions/CN=localhost"
  cat pgroute66.crt pgroute66.key
  CERT=$(base64 -w0 < pgroute66.crt)
  KEY=$(base64 -w0 < pgroute66.key)
  echo -e "ssl:\n  b64cert: ${CERT}\n  b64key: ${KEY}" >> ./test/config.yaml
  cat ./test/config.yaml
else
  echo "reusing existing ./test/config.yaml"
fi

docker-compose down && docker rmi pgroute66-postgres pgroute66-pgroute66  || echo new install
docker-compose up -d --scale postgres=3
for ((i=1;i<=3;i++)); do
  docker exec "pgroute66-postgres-${i}" /entrypoint.sh background
done

docker-compose up -d pgroute66
docker ps -a
assert primary 'host1'
assert primaries '[ host1 ]'
assert standbys '[ host2, host3 ]'

docker exec pgroute66-postgres-2 /entrypoint.sh promote
assert primary ''
assert primaries '[ host1, host2 ]'
assert standbys '[ host3 ]'

docker exec pgroute66-postgres-1 /entrypoint.sh rebuild
assert primary 'host2'
assert primaries '[ host2 ]'
assert standbys '[ host1, host3 ]'

echo "All is as expected"
