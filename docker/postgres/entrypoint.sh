#!/bin/bash
set -e

PGBIN=${PGBIN:-/usr/pgsql-${PGVERSION}/bin}
export PATH="${PGBIN}:$PATH"
export PGDATA=${PGDATA:-/var/lib/pgsql/${PGVERSION}/data}

function primary() {
  PGVERSION=${PGVERSION:-12}
  if [ ! -e "${PGDATA}" ]; then
    mkdir -p "${PGDATA}"
    chown postgres: "${PGDATA}"
  fi

  if [ ! -e "${PGDATA}/PG_VERSION" ]; then
    PWFILE=$(mktemp)
    echo "${PGPASSWORD}" > "${PWFILE}"
    initdb --pwfile="${PWFILE}" || return $?
    rm "${PWFILE}"
    mkdir "${PGDATA}/conf.d"
    echo "include_dir 'conf.d'" >> "${PGDATA}/postgresql.conf"
    echo "listen_addresses = '*'" >> "${PGDATA}/conf.d/listen_address.conf"
    while read IP
    do
      echo "
host    all             postgres        ${IP}               md5
host    replication     postgres        ${IP}               md5" >> "${PGDATA}/pg_hba.conf"

      re="^([0-9.]+)(.*)$"
      if [[ $IP =~ $re ]]; then
        [ -n ${BASH_REMATCH[1]} -a ${BASH_REMATCH[1]} != '127.0.0.1' ] && MYIP=${BASH_REMATCH[1]}
      fi
    done <<< "$(ip a | sed -n '/inet /{s/.*inet //;s/ .*//;p}')"
  fi
}

function standby() {
  PGVERSION=${PGVERSION:-12}
  PGTARGET_SESSION_ATTRS=read-write
  export PGHOST=${PGHOSTS}
  if [ ! -e "${PGDATA}" ]; then
    mkdir -p "${PGDATA}"
    chown postgres: "${PGDATA}"
  fi

  if [ ! -e "${PGDATA}/PG_VERSION" ]; then
    pg_basebackup -R -D "${PGDATA}" || return $?
    chmod 0700 "${PGDATA}"
  fi
}

function pg_start() {
  postgres -D "${PGDATA}"
}

function pg_start_bg() {
  pg_ctl start -D "${PGDATA}" -l "${PGDATA}/postgres.log"
}

function pg_stop_bg() {
  pg_ctl stop -D "${PGDATA}" -l "${PGDATA}/postgres.log"
}

function pg_promote() {
  psql -c 'select pg_promote();'
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
  rebuild)
    pg_stop_bg
    rm -rf "${PGDATA}"/*
    standby
    pg_start_bg
    ;;
  background)
    standby || primary
    pg_start_bg
    ;;
  promote)
    pg_promote
    ;;
  auto)
    standby || primary
    pg_start
    ;;
  sleep)
     waitsleep
     ;;
esac
