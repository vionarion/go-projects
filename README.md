# go-projects

![ci](https://github.com/vionarion/go-projects/actions/workflows/ci.yml/badge.svg)
![license](https://img.shields.io/github/license/vionarion/go-projects)

Go projects: from a standard-library HTTP API to a containerized service with Redis.
Every project has tests; CI runs `gofmt`, `go vet`, tests with the race detector,
security checks (govulncheck, gosec, Trivy for the image and the Helm chart, gitleaks)
and a Docker image build on every push. url-shortener releases are signed with cosign
and ship an SBOM; a Kyverno policy admits only signed images to the cluster.
In Kubernetes the Redis password comes from HashiCorp Vault (HA, Raft, TLS) through
Vault Secrets Operator and rotates without downtime.

| Project | What it is |
|---|---|
| [url-shortener](url-shortener/) | Short links service: Fiber + Redis, custom codes, link expiry, per-IP rate limit, healthcheck, distroless image, Docker Compose |
| [simple-web](simple-web/) | JSON API on `net/http` only (Go 1.22+ method routing), concurrency-safe in-memory storage |
| [word-game](word-game/) | CLI helper for a word party game: random letter + 4 distinct categories |
| [netkit](netkit/) | CLI with sysinfo and a concurrent TCP port scanner, standard library only |

## Quick start

```bash
cd url-shortener && cp .env.example .env && docker compose up -d --build
cd simple-web && go run .
cd word-game && go run .
cd netkit && go run . sysinfo
```

Each project is a separate Go module, so it can be built and tested on its own.
Secrets live only in local `.env` files, which are ignored by git.
