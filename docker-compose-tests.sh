#!/bin/bash
TST=0
function assert() {
  TST=$((TST+1))
  EP=$1
  EXPECTED=$2
  RESULT=$(curl "http://localhost:8080/v1/${EP}" 2>/dev/null | xargs)
  if [ "${RESULT}" = "${EXPECTED}" ]; then
    echo "test${TST}: OK"
  else
    echo "test${TST}: EROR: expected '${EXPECTED}', but got '${RESULT}'"
    return 1
  fi
}

#set -x
set -e
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
