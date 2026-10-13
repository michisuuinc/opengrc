#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Builds the probod image from this fork (linux/amd64) on a workstation,
# pushes it to ECR and records the image by digest (repo@sha256:...) in
# SSM (/opengrc/image) for deploy.sh. Tags in ECR are immutable, and the
# digest guarantees the host runs exactly the image built here.
# Mirrors .github/workflows/release-probod.yaml.
#
# Requires: Go and Node versions from go.mod / .nvmrc, Docker with buildx,
# AWS CLI credentials. Run from anywhere; the working tree must be clean.

set -euo pipefail

REGION="${AWS_REGION:-ap-northeast-1}"
ECR_REPO="opengrc/probod"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export AWS_PAGER=""

cd "$ROOT"
[[ -z "$(git status --porcelain)" ]] || { echo "error: working tree is not clean" >&2; exit 1; }

sha="$(git rev-parse --short=12 HEAD)"
version="$(cat cmd/probod/VERSION)"
account="$(aws sts get-caller-identity --query Account --output text)"
registry="$account.dkr.ecr.$REGION.amazonaws.com"
image="$registry/$ECR_REPO:$version-$sha"

echo "==> building web apps"
git submodule update --init --recursive
npm ci
npm --workspace @probo/emails run build
make relay
for app in console compliance-portal employee-portal; do
  NODE_ENV=production npm --workspace "@probo/$app" run build
done

echo "==> building linux/amd64 binaries"
make generate
mkdir -p linux/amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-s -w -X main.version=$version -X main.env=prod" \
  -o linux/amd64/probod ./cmd/probod/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-s -w" -o linux/amd64/probod-bootstrap ./cmd/probod-bootstrap/main.go

echo "==> building and pushing $image"
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$registry"
docker buildx build --platform linux/amd64 --provenance=false -t "$image" --push .

digest="$(aws ecr describe-images --region "$REGION" --repository-name "$ECR_REPO" \
  --image-ids "imageTag=$version-$sha" --query 'imageDetails[0].imageDigest' --output text)"
[[ "$digest" == sha256:* ]] || { echo "error: cannot read the pushed image digest" >&2; exit 1; }
pinned="$registry/$ECR_REPO@$digest"

aws ssm put-parameter --region "$REGION" --name /opengrc/image --type String \
  --value "$pinned" --overwrite >/dev/null
rm -rf linux
echo "==> /opengrc/image = $pinned ($version-$sha)"
