#!/bin/sh
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Runs once, when the Postgres data volume is empty. The probod role owns
# its database but is not a superuser: citext, pgcrypto and unaccent are
# trusted extensions the owner can create during migrations, and only
# pg_stat_statements (untrusted) is created here.

set -eu

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" \
  -v probod_password="$PROBOD_PG_PASSWORD" <<-'EOF'
CREATE ROLE probod LOGIN PASSWORD :'probod_password';
CREATE DATABASE probod OWNER probod;
EOF

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d probod \
  -v base_domain="$TRUST_CENTER_BASE_DOMAIN" <<-'EOF'
ALTER SCHEMA public OWNER TO probod;
GRANT ALL ON SCHEMA public TO probod;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
ALTER DATABASE probod SET probo.trust_center_base_domain TO :'base_domain';
EOF
