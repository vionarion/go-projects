#!/usr/bin/env bash
# Configures Vault for url-shortener: audit log, KV v2, Kubernetes auth, least-privilege policy.
# Idempotent. Needs vault-init.json from init.sh (uses the root token only for this setup;
# in production: an admin role via OIDC, root token revoked after bootstrap).
set -euo pipefail
NS=vault
TOKEN=$(jq -r .root_token vault-init.json)
vx() { kubectl -n "$NS" exec -i vault-0 -- env VAULT_TOKEN="$TOKEN" vault "$@"; }

vx audit list | grep -q '^file/' || vx audit enable file file_path=/vault/audit/audit.log
vx secrets list | grep -q '^secret/' || vx secrets enable -path=secret -version=2 kv

# The Redis password is generated here and never leaves Vault except through the operator.
vx kv get secret/urls/redis >/dev/null 2>&1 || \
  vx kv put secret/urls/redis password="$(openssl rand -hex 24)"

vx auth list | grep -q '^kubernetes/' || vx auth enable kubernetes
# Vault validates pod tokens with the TokenReview API using its own service account
# (chart value server.authDelegator.enabled grants system:auth-delegator).
vx write auth/kubernetes/config kubernetes_host="https://kubernetes.default.svc:443"

vx policy write urls-redis - <<'HCL'
# read-only, one path
path "secret/data/urls/redis" {
  capabilities = ["read"]
}
HCL

vx write auth/kubernetes/role/urls \
  bound_service_account_names=vault-auth \
  bound_service_account_namespaces=urls \
  audience=vault \
  token_policies=urls-redis \
  token_ttl=1h token_max_ttl=4h
