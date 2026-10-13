#!/bin/bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# EC2 first-boot script (cloud-init user data), rendered by provision.sh.
# Removes SSH entirely (access is through SSM Session Manager only),
# installs Docker and the AWS CLI, checks out the deploy files at the
# commit given by provision.sh, restricts instance metadata access and
# installs the daily database backup timer. It does not start the stack:
# deploy.sh does. On success it writes /opt/opengrc/.bootstrap-complete,
# which deploy.sh checks.

set -euo pipefail

REPO_URL="@REPO_URL@"
REPO_SHA="@REPO_SHA@"
APP_DIR="/opt/opengrc"

export DEBIAN_FRONTEND=noninteractive

# No SSH, before anything else: no key pair is attached and port 22 is
# closed in the security group; the server and EC2 Instance Connect are
# removed and the units masked so no upgrade can bring them back.
systemctl disable --now ssh.service ssh.socket || true
systemctl mask ssh.service ssh.socket
apt-get purge -y openssh-server ec2-instance-connect || true

apt-get update
apt-get upgrade -y
# fonts-noto-cjk: the headless Chrome image has no Japanese fonts; they are
# mounted into the chrome container (compose.yaml) so PDFs render Japanese.
apt-get install -y docker.io docker-compose-v2 git unattended-upgrades fonts-noto-cjk
systemctl enable --now docker
snap install aws-cli --classic

# Security updates are applied automatically (Ubuntu default, made explicit).
cat >/etc/apt/apt.conf.d/20auto-upgrades <<'CONF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
CONF

mkdir -p "$APP_DIR"
git init -q "$APP_DIR/src"
git -C "$APP_DIR/src" remote add origin "$REPO_URL"
git -C "$APP_DIR/src" fetch -q --depth 1 origin "$REPO_SHA"
git -C "$APP_DIR/src" checkout -q --force "$REPO_SHA"
[ "$(git -C "$APP_DIR/src" rev-parse HEAD)" = "$REPO_SHA" ]

for unit in opengrc-imds-guard.service opengrc-backup.service opengrc-backup.timer; do
  install -m 0644 "$APP_DIR/src/deploy/aws-ec2/systemd/$unit" /etc/systemd/system/
done
systemctl daemon-reload
systemctl enable --now opengrc-imds-guard.service
systemctl enable opengrc-backup.timer

date -u +%Y-%m-%dT%H:%M:%SZ >"$APP_DIR/.bootstrap-complete"
