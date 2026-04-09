#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${1:-demo-restored}"
DEPLOYMENT="${2:-app}"
FILE_PATH="${3:-/usr/share/nginx/html/data/check.txt}"

kubectl -n "${NAMESPACE}" rollout status deploy/"${DEPLOYMENT}" --timeout=180s
kubectl -n "${NAMESPACE}" get all
kubectl -n "${NAMESPACE}" get pvc -o wide
kubectl -n "${NAMESPACE}" exec deploy/"${DEPLOYMENT}" -- sh -c "cat '${FILE_PATH}'"
