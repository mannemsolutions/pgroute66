#!/bin/bash
set -x
set -e
docker-compose up -d --scale postgres=3
for ((i=1;i<4;i++)); do
  docker exec -ti pgroute66_postgres_$i /entrypoint.sh background
done

docker-compose up pgfga || exit 2
cat testdata/pgtester/tests.yaml | docker-compose run pgtester pgtester || exit 3

echo "All is as expected"
