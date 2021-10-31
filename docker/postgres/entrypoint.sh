#!/bin/bash
set -e

PGBIN=${PGBIN:-/usr/pgsql-${PGVERSION}/bin}
export PATH="${PGBIN}:$PATH"

function primary() {
  PGVERSION=${PGVERSION:-12}
  export PGDATA=${PGDATA:-/var/lib/pgsql/${PGVERSION}/data}
  if [ ! -e "${PGDATA}" ]; then
    mkdir -p "${PGDATA}"
    chown postgres: "${PGDATA}"
  fi

  re="^([0-9.]+)(.*)$"
  
  [ -e "${PGDATA}/PG_VERSION" ] || initdb
  if [ ! -e "${PGDATA}/conf.d" ]; then
    mkdir "${PGDATA}/conf.d"
    echo "include_dir 'conf.d'" >> "${PGDATA}/postgresql.conf"
    echo "listen_addresses = '*'" >> "${PGDATA}/conf.d/listen_address.conf"
  fi
  
  postgres -D "${PGDATA}"
}

function standby() {
  PGVERSION=${PGVERSION:-12}
  export PGDATA=${PGDATA:-/var/lib/pgsql/${PGVERSION}/data}
  if [ ! -e "${PGDATA}" ]; then
    mkdir -p "${PGDATA}"
    chown postgres: "${PGDATA}"
  fi

  [ -e "${PGDATA}/PG_VERSION" ] || pg_basebackup -R -D "${PGDATA}"
  postgres -D "${PGDATA}"
}

function waitsleep() {
  SLEEPTIME=${SLEEPTIME:-10}
  while /bin/sleep ${SLEEPTIME}; do
    echo "$(date "+%Y-%m-%d %H:%M:%S") sleep ${SLEEPTIME}"
  done
}

case "${1}" in
  primary)
    primary
    ;;
  standby)
    standby
    ;;
  auto)
    standby || primary
    ;;
  sleep)
     waitsleep
     ;;
esac
