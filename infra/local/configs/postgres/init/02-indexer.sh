#!/bin/sh
# Creates the indexer's role and its databases: the one the service uses and
# an empty one for the integration tests, so they never touch real data.
set -eu

psql -v ON_ERROR_STOP=1 \
     -v password="$INDEXER_DB_PASSWORD" \
     --username "$POSTGRES_USER" --dbname postgres <<'SQL'
CREATE ROLE indexer LOGIN PASSWORD :'password';
CREATE DATABASE ecoa_indexer OWNER indexer;
REVOKE CONNECT ON DATABASE ecoa_indexer FROM PUBLIC;

CREATE DATABASE ecoa_indexer_test OWNER indexer;
REVOKE CONNECT ON DATABASE ecoa_indexer_test FROM PUBLIC;
SQL
