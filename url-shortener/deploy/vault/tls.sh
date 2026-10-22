#!/usr/bin/env bash
# Creates a private CA and a server certificate for Vault, stores them as Secret vault/vault-tls
# and puts the CA certificate into the app namespace for the operator (Secret urls/vault-ca).
# The CA key stays in ./pki (git-ignored). In production: cert-manager or the company PKI.
set -euo pipefail
NS=vault APP_NS=urls DIR=pki
mkdir -p "$DIR" && cd "$DIR"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 365 \
  -keyout ca.key -out ca.crt -subj "/CN=Vault lab CA"

openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout tls.key -out tls.csr -subj "/CN=vault.$NS.svc"

cat > san.ext <<EXT
subjectAltName = DNS:vault, DNS:vault.$NS, DNS:vault.$NS.svc, DNS:vault.$NS.svc.cluster.local, DNS:vault-active.$NS.svc, DNS:*.vault-internal, DNS:*.vault-internal.$NS.svc.cluster.local, DNS:localhost, IP:127.0.0.1
extendedKeyUsage = serverAuth, clientAuth
EXT
openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key -CAcreateserial -days 90 \
  -extfile san.ext -out tls.crt
chmod 600 ca.key tls.key

kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$NS" create secret generic vault-tls \
  --from-file=ca.crt --from-file=tls.crt --from-file=tls.key \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$APP_NS" create secret generic vault-ca --from-file=ca.crt \
  --dry-run=client -o yaml | kubectl apply -f -
