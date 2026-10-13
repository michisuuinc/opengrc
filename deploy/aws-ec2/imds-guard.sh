#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Runs on the EC2 host after Docker starts (opengrc-imds-guard.service).
# Blocks every container from the instance metadata service except probo's
# fixed addresses in compose.yaml, so only probod can obtain the instance
# role credentials. The host itself is not affected (DOCKER-USER only sees
# forwarded container traffic).

set -euo pipefail

IMDS="169.254.169.254"
PROBO_ADDRS=("172.30.0.10" "172.30.2.10")

# Idempotent: remove our previous rules, then insert them again.
while iptables -D DOCKER-USER -d "$IMDS" -j DROP 2>/dev/null; do :; done
for addr in "${PROBO_ADDRS[@]}"; do
  while iptables -D DOCKER-USER -s "$addr" -d "$IMDS" -j ACCEPT 2>/dev/null; do :; done
done

iptables -I DOCKER-USER -d "$IMDS" -j DROP
for addr in "${PROBO_ADDRS[@]}"; do
  iptables -I DOCKER-USER -s "$addr" -d "$IMDS" -j ACCEPT
done
echo "IMDS restricted to ${PROBO_ADDRS[*]}"
