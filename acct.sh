#!/usr/bin/env bash
# acct.sh — Account management wrapper (disable / enable / revive)
#
# Usage:
#   ./acct.sh list                     # List accounts and both disable flags
#   ./acct.sh disable <uid> [reason]     # Remove from chat selection while keeping it in the pool
#   ./acct.sh enable  <uid>            # Clear manual disabling
#   ./acct.sh revive  <uid>            # Clear automatic disabling
#
# Requires admin.enabled = true and a nonempty api_key. Arguments pass through to cmd/acct;
# -config and -server are supported: ./acct.sh -server http://127.0.0.1:7863 list
#
# Build binary: go build -o acct ./cmd/acct
set -euo pipefail
cd "$(dirname "$0")"
exec go run ./cmd/acct "$@"
