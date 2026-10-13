#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Runs on the EC2 host (deploy.sh calls it). Writes runtime.conf with the
# image reference and the Postgres passwords read from SSM through the
# instance role, readable by root only, and logs Docker in to ECR.
# probod's own secrets are not copied here: probod-bootstrap reads them
# from SSM inside the container.

set -euo pipefail

REGION="ap-northeast-1"
SSM_PREFIX="/opengrc"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$DIR/runtime.conf"

get() {
  aws ssm get-parameter --region "$REGION" --with-decryption \
    --name "$SSM_PREFIX/$1" --query Parameter.Value --output text
}

image="$(get image)"
umask 077
{
  printf 'PROBOD_IMAGE=%s\n' "$image"
  printf 'POSTGRES_PASSWORD=%s\n' "$(get postgres-password)"
  printf 'PROBOD_PG_PASSWORD=%s\n' "$(get probod-pg-password)"
} >"$OUT.tmp"
mv "$OUT.tmp" "$OUT"

registry="${image%%/*}"
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$registry" >/dev/null
echo "runtime.conf written for $image"
