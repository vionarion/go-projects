#!/usr/bin/env bash
# Rotates the Redis password: new random value, the old one kept as "previous".
# The operator syncs it within refreshAfter (30 s), Redis accepts both, url-shortener rolls.
set -euo pipefail
TOKEN=$(jq -r .root_token vault-init.json)
vx() { kubectl -n vault exec -i vault-0 -- env VAULT_TOKEN="$TOKEN" vault "$@"; }
old=$(vx kv get -field=password secret/urls/redis)
vx kv put secret/urls/redis password="$(openssl rand -hex 24)" previous="$old" >/dev/null
echo "rotated, version $(vx kv metadata get -format=json secret/urls/redis | jq .data.current_version)"
