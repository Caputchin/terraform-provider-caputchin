#!/usr/bin/env bash
#
# testacc-bootstrap.sh: run the provider acceptance tests against a running
# Caputchin management API.
#
# Acceptance tests (TF_ACC) drive REAL terraform plan/apply/import/destroy
# against a LIVE API, so they create and destroy real resources. They never
# run in CI (TF_ACC is unset there, so resource.Test skips). This helper
# validates the prerequisites before handing off to `make testacc`:
#
#   CAPUTCHIN_ENDPOINT         Required. Management API base URL. NOT defaulted,
#                              so the suite can never run against production by
#                              accident; point it at a non-production instance.
#   CAPUTCHIN_MANAGEMENT_TOKEN Required. An account-PAT. PATs cannot be minted
#                              non-interactively; create one in the dashboard.
#
# Usage:
#   export CAPUTCHIN_ENDPOINT=...           # the API base for a test instance
#   export CAPUTCHIN_MANAGEMENT_TOKEN=...   # an account-PAT
#   ./scripts/testacc-bootstrap.sh                              # whole suite
#   ./scripts/testacc-bootstrap.sh -run TestAccCustomizedGame   # one test
#
set -euo pipefail

ENDPOINT="${CAPUTCHIN_ENDPOINT:-}"
TOKEN="${CAPUTCHIN_MANAGEMENT_TOKEN:-}"

if [[ -z "${ENDPOINT}" ]]; then
  echo "ERROR: CAPUTCHIN_ENDPOINT is unset. Point it at a NON-PRODUCTION" >&2
  echo "  management API base URL (acceptance tests create + destroy data)." >&2
  exit 1
fi
if [[ -z "${TOKEN}" ]]; then
  echo "ERROR: CAPUTCHIN_MANAGEMENT_TOKEN is unset. Create an account-PAT in" >&2
  echo "  the dashboard (Settings -> Tokens) and export it." >&2
  exit 1
fi

# Preflight: confirm the API is reachable AND the token authenticates, so a
# down instance or a bad token fails here with a clear message instead of deep
# inside a terraform apply.
echo "==> preflight: GET ${ENDPOINT}/v1/management/me/account"
code="$(curl -s -o /dev/null -m 10 -w '%{http_code}' \
  -H "Authorization: Bearer ${TOKEN}" \
  "${ENDPOINT}/v1/management/me/account" || true)"

case "${code}" in
  200) echo "==> API reachable, token valid (HTTP 200)" ;;
  000) echo "ERROR: no response from ${ENDPOINT}. Is the API up and the URL correct?" >&2; exit 1 ;;
  401 | 403) echo "ERROR: token rejected (HTTP ${code}). Check CAPUTCHIN_MANAGEMENT_TOKEN." >&2; exit 1 ;;
  *) echo "ERROR: unexpected HTTP ${code} from ${ENDPOINT}/v1/management/me/account." >&2; exit 1 ;;
esac

cd "$(dirname "$0")/.."

# A subset run (args present) goes straight to `go test` so -run/-v pass
# through; a bare invocation uses the documented `make testacc` entry point.
if [[ $# -gt 0 ]]; then
  echo "==> go test (TF_ACC=1) ${*}"
  exec env TF_ACC=1 go test ./internal/provider/... -v -timeout=300s "$@"
fi

echo "==> make testacc"
exec env TF_ACC=1 make testacc
