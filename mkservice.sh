#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./mkservice.sh <service_name> [module_path]
#
# Examples:
#   ./mkservice.sh payments github.com/mitrich772/L3.2/services/payments
#   ./mkservice.sh analytics  github.com/you/repo/services/analytics
#
# If module_path is omitted -> example.com/<service_name>

SERVICE="${1:-}"
MODULE_PATH="${2:-}"

if [[ -z "$SERVICE" ]]; then
  echo "Usage: $0 <service_name> [module_path]"
  exit 1
fi

if [[ -z "$MODULE_PATH" ]]; then
  MODULE_PATH="${SERVICE}"
fi

ROOT_DIR="services"
SVC_DIR="${ROOT_DIR}/${SERVICE}"

if [[ -e "$SVC_DIR" ]]; then
  echo "Error: '${SVC_DIR}' already exists"
  exit 1
fi

# Create directories
mkdir -p \
  "${SVC_DIR}/cmd/${SERVICE}" \
  "${SVC_DIR}/config" \
  "${SVC_DIR}/internal/cache" \
  "${SVC_DIR}/internal/config" \
  "${SVC_DIR}/internal/handlers" \
  "${SVC_DIR}/internal/middleware/logger" \
  "${SVC_DIR}/internal/service" \
  "${SVC_DIR}/internal/store" \
  "${SVC_DIR}/migrations" \
  "${SVC_DIR}/web"

# main.go (minimal web service)
cat > "${SVC_DIR}/cmd/${SERVICE}/main.go" <<'EOF'
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := getenv("ADDR", ":8080")
	log.Printf("listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
EOF

# Sample config
cat > "${SVC_DIR}/config/local.yaml" <<'EOF'
server:
  addr: ":8080"
db:
  dsn: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
EOF

# Migrations placeholders
cat > "${SVC_DIR}/migrations/0001_init.up.sql" <<'EOF'
-- put your schema here
-- example:
-- CREATE TABLE example (id BIGSERIAL PRIMARY KEY);
EOF

cat > "${SVC_DIR}/migrations/0001_init.down.sql" <<'EOF'
-- rollback for 0001_init.up.sql
-- example:
-- DROP TABLE IF EXISTS example;
EOF

# Web placeholder (swagger/openapi/static)
cat > "${SVC_DIR}/web/.gitkeep" <<'EOF'
EOF

# README
cat > "${SVC_DIR}/README.md" <<EOF
# ${SERVICE}

## Run
\`\`\`bash
cd ${SVC_DIR}
go run ./cmd/${SERVICE}
\`\`\`

## Healthcheck
- GET /health

## Config
- \`config/config.local.yaml\` — пример локального конфига
EOF

# Ensure dirs are tracked even if empty
for d in \
  "${SVC_DIR}/internal/cache" \
  "${SVC_DIR}/internal/config" \
  "${SVC_DIR}/internal/handlers" \
  "${SVC_DIR}/internal/middleware/logger" \
  "${SVC_DIR}/internal/service" \
  "${SVC_DIR}/internal/store"
do
  : > "${d}/.gitkeep"
done

# Init go module (if go installed)
if command -v go >/dev/null 2>&1; then
  (
    cd "${SVC_DIR}"
    go mod init "${MODULE_PATH}" >/dev/null
    go mod tidy >/dev/null
  )
  echo "Created ${SVC_DIR} (module: ${MODULE_PATH})"
else
  echo "Created ${SVC_DIR} (go not found, skipped go mod init)"
fi
