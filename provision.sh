#!/bin/sh
# cpus: 2
# portfwd: 5432
# portfwd: 6379

_psql() {
	sudo -u postgres psql -c "$1"
}

if [ -f .provisioned ]; then
	exit 0
fi

export PGPASSWORD="secret"

_psql "DROP DATABASE IF EXISTS djinn;"
_psql "DROP USER IF EXISTS djinn_consumer, djinn_curator, djinn_scheduler, djinn_server, djinn_worker;"
_psql "CREATE DATABASE djinn;"

keys="$(redis-cli KEYS \*)"

if [ ! -z "$keys" ]; then
	redis-cli DEL $keys
fi

touch .provisioned
