#!/usr/bin/env bash
# Detect PR domains from pr_diff.patch using keyword matching.
# Output: JSON array e.g. ["sql", "auth", "async"]
# Deterministic — zero LLM calls.
set -euo pipefail

DIFF_FILE="pr_diff.patch"
[ -n "${CORPUS_ENTRY_DIR:-}" ] && DIFF_FILE="${CORPUS_ENTRY_DIR}/pr_diff.patch"

if [ ! -f "$DIFF_FILE" ]; then
  echo '[]'
  exit 0
fi

domains=()

{ grep -qiE '(SELECT|INSERT|UPDATE|DELETE|CREATE TABLE|ALTER TABLE|DROP TABLE|JOIN|WHERE|FROM\s|sql|query|migration|schema|index|transaction|cursor|prepared.statement|ORM|sequelize|prisma|hibernate|sqlalchemy)' "$DIFF_FILE" && domains+=("sql"); } || true
{ grep -qiE '(auth|token|jwt|session|oauth|password|credential|permission|role|acl|cookie|bearer|secret|api.?key|csrf|cors|login|logout|signup|register|authenticate|authorize)' "$DIFF_FILE" && domains+=("auth"); } || true
{ grep -qiE '(async|await|Promise|goroutine|channel|mutex|lock|race|concurrent|thread|coroutine|semaphore|WaitGroup|sync\.|atomic|callback|EventLoop|setTimeout|setInterval|TaskGroup|asyncio)' "$DIFF_FILE" && domains+=("async"); } || true

printf '%s\n' "${domains[@]}" | jq -Rn '[inputs]'
