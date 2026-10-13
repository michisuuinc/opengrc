#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Runs on the EC2 host (systemd timer, daily). Streams a pg_dump of the
# probod database to S3; the bucket lifecycle deletes dumps after 30 days.

set -euo pipefail

BUCKET="michisuu-opengrc-backup"
REGION="ap-northeast-1"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
key="db/probod-$(date -u +%Y%m%dT%H%M%SZ).dump"

cd "$DIR"
docker compose --env-file runtime.conf exec -T postgres \
  pg_dump -U postgres -d probod -Fc \
  | aws s3 cp - "s3://$BUCKET/$key" --region "$REGION" --sse AES256
echo "backup written to s3://$BUCKET/$key"
