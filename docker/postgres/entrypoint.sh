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
  
  if [ ! -e "${PGDATA}/PG_VERSION" ]; then
    PWFILE=$(mktemp)
    echo "${PGPASSWORD}" > "${PWFILE}"
    initdb --pwfile="${PWFILE}"
    rm "${PWFILE}"
    mkdir "${PGDATA}/conf.d"
    echo "include_dir 'conf.d'" >> "${PGDATA}/postgresql.conf"
    echo "listen_addresses = '*'" >> "${PGDATA}/conf.d/listen_address.conf"
    while read IP
    do
      echo "
host    all             postgres        ${IP}               md5
host    replication     postgres        ${IP}               md5" >> "${PGDATA}/pg_hba.conf"
      if [[ $IP =~ $re ]]; then
        [ -n ${BASH_REMATCH[1]} -a ${BASH_REMATCH[1]} != '127.0.0.1' ] && MYIP=${BASH_REMATCH[1]}
      fi
    done <<< "$(ip a | sed -n '/inet /{s/.*inet //;s/ .*//;p}')"
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
  chmod 0700 "${PGDATA}"
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
