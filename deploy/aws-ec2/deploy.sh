#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Deploys to the EC2 host through SSM Run Command (no SSH).
#
#   ./deploy.sh [REF]        # default: origin/main
#
# REF is resolved to a full commit SHA on this workstation and must already
# be part of origin/main (reviewed and merged). The host fetches exactly
# that SHA, verifies it, refreshes runtime.conf from SSM, pulls the image
# and restarts the stack. Anything pushed to the public repository that is
# not on main cannot reach the host.

set -euo pipefail

REGION="${AWS_REGION:-ap-northeast-1}"
REF="${1:-origin/main}"
export AWS_PAGER=""

[[ "$REF" =~ ^[A-Za-z0-9._/][A-Za-z0-9._/-]*$ && "$REF" != *..* ]] \
  || { echo "error: invalid ref" >&2; exit 1; }

git fetch -q origin main
sha="$(git rev-parse --verify --quiet "$REF^{commit}")" \
  || { echo "error: unknown ref $REF" >&2; exit 1; }
git merge-base --is-ancestor "$sha" origin/main \
  || { echo "error: $sha is not on origin/main; merge it through a reviewed PR first" >&2; exit 1; }

instance="$(aws ec2 describe-instances --region "$REGION" \
  --filters Name=tag:Name,Values=opengrc Name=instance-state-name,Values=running \
  --query 'Reservations[0].Instances[0].InstanceId' --output text)"
[[ "$instance" != "None" ]] || { echo "error: no running opengrc instance" >&2; exit 1; }

script="set -euo pipefail
test -f /opt/opengrc/.bootstrap-complete || { echo 'first-boot setup did not finish; check /var/log/cloud-init-output.log' >&2; exit 1; }
cd /opt/opengrc/src
git fetch -q --depth 1 origin $sha
git checkout -q --force $sha
test \"\$(git rev-parse HEAD)\" = $sha
for unit in opengrc-imds-guard.service opengrc-backup.service opengrc-backup.timer; do
  install -m 0644 deploy/aws-ec2/systemd/\$unit /etc/systemd/system/
done
systemctl daemon-reload
systemctl restart opengrc-imds-guard.service
cd deploy/aws-ec2
./fetch-secrets.sh
docker compose --env-file runtime.conf pull
docker compose --env-file runtime.conf up -d --remove-orphans
docker image prune -f
docker compose --env-file runtime.conf ps"

params="$(node -e 'process.stdout.write(JSON.stringify({commands: ["bash -c " + JSON.stringify(process.argv[1])]}))' "$script")"

command_id="$(aws ssm send-command --region "$REGION" --instance-ids "$instance" \
  --document-name AWS-RunShellScript --comment "opengrc deploy $sha" \
  --parameters "$params" --query Command.CommandId --output text)"

echo "==> deploying $sha to $instance (command $command_id)"
aws ssm wait command-executed --region "$REGION" --command-id "$command_id" --instance-id "$instance" || true
aws ssm get-command-invocation --region "$REGION" --command-id "$command_id" --instance-id "$instance" \
  --query '{status: Status, output: StandardOutputContent, error: StandardErrorContent}' --output text
