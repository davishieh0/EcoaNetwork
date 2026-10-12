#!/bin/sh
# Creates the backend's role and its databases: the one the service uses and
# an empty one for the integration tests, so they never touch real data.
set -eu

psql -v ON_ERROR_STOP=1 \
     -v password="$BACKEND_DB_PASSWORD" \
     --username "$POSTGRES_USER" --dbname postgres <<'SQL'
CREATE ROLE backend LOGIN PASSWORD :'password';
CREATE DATABASE ecoa_backend OWNER backend;
REVOKE CONNECT ON DATABASE ecoa_backend FROM PUBLIC;

CREATE DATABASE ecoa_backend_test OWNER backend;
REVOKE CONNECT ON DATABASE ecoa_backend_test FROM PUBLIC;
SQL
