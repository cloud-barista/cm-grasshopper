#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${1:-demo}"
DEPLOYMENT="${2:-app}"
FILE_PATH="${3:-/usr/share/nginx/html/data/check.txt}"
VALUE="${4:-hello-velero}"

kubectl -n "${NAMESPACE}" rollout status deploy/"${DEPLOYMENT}" --timeout=120s
kubectl -n "${NAMESPACE}" exec deploy/"${DEPLOYMENT}" -- sh -c "echo '${VALUE}' > '${FILE_PATH}'"
kubectl -n "${NAMESPACE}" exec deploy/"${DEPLOYMENT}" -- sh -c "cat '${FILE_PATH}'"
