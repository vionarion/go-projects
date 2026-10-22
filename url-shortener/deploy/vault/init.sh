#!/usr/bin/env bash
# Initializes the Raft cluster once and unseals all three pods.
# Shamir: 5 key shares, any 3 unseal. The output (keys + root token) goes to ./vault-init.json,
# mode 600, git-ignored. In production: auto-unseal with a cloud KMS (AWS KMS, Alibaba KMS)
# or the Transit engine of another Vault, and the key shares split between people.
set -euo pipefail
NS=vault OUT=vault-init.json
v() { kubectl -n "$NS" exec "$1" -- vault "${@:2}"; }
# "vault status" exits with 2 while sealed, so read its JSON and ignore the code
st() { v "$1" status -format=json 2>/dev/null || true; }

if [ "$(st vault-0 | jq -r .initialized)" != "true" ]; then
  [ -e "$OUT" ] && { echo "$OUT exists but Vault is not initialized, stop"; exit 1; }
  umask 077
  v vault-0 operator init -key-shares=5 -key-threshold=3 -format=json > "$OUT"
  echo "initialized, keys saved to $OUT"
fi

for pod in vault-0 vault-1 vault-2; do
  # standby pods join the leader through retry_join; wait until they see it
  for _ in $(seq 30); do
    [ "$(st "$pod" | jq -r .sealed)" = "false" ] && break
    for i in 0 1 2; do
      v "$pod" operator unseal "$(jq -r ".unseal_keys_b64[$i]" "$OUT")" >/dev/null || true
    done
    sleep 2
  done
  echo "$pod: sealed=$(st "$pod" | jq -r .sealed)"
done

VAULT_TOKEN=$(jq -r .root_token "$OUT")
kubectl -n "$NS" exec vault-0 -- env VAULT_TOKEN="$VAULT_TOKEN" VAULT_ADDR=https://vault-active.vault.svc:8200 \
  vault operator raft list-peers
