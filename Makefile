.PHONY: all build-ui build-go build clean run demo install test bench bench-file \
	bench-extensive bench-baseline stress load-test test-load perf perf-seed \
	perf-report perf-baseline perf-gate perf-clean help

# Config
BINARY_NAME=rss-cli
PORT?=7077
DATA_DIR?=./data
SILENT?=true

# Dataset de rendimiento. PERF_DIR es siempre un directorio aparte: el seed de
# perf nunca toca $(DATA_DIR) de desarrollo.
PERF_DIR?=./data/perf
PERF_PORT?=17077
SEED_N?=10000
PERF_LEVELS?=10,50,100,200
PERF_DURATION?=10s
PERF_WARMUP?=3s
PERF_PROFILES?=health,news,feeds,search,search1,write

# Demo para capturas: usuario + subset curado de 8 feeds con imagen,
# sincronizados de verdad (con sus fotos) + filtros de ejemplo, en un
# directorio aparte del de desarrollo y del de rendimiento.
DEMO_DIR?=./data/demo
DEMO_PORT?=7077
DEMO_USER?=demo
DEMO_PASS?=DemoPass123!
DEMO_SECRET?=gorrson-demo-jwt-secret-at-least-32-chars

# Default target: Show help
help:
	@echo "Gorrson Management:"
	@echo "  make install    - Install all dependencies (npm & go)"
	@echo "  make build      - Build full app (Frontend + Backend)"
	@echo "  make run        - Build and start the production server"
	@echo "  make dev-ui     - Start Svelte/Vite dev server"
	@echo "  make dev-go     - Start Go backend in dev mode"
	@echo "  make dev        - Start frontend & backend in parallel dev mode"
	@echo "  make demo       - Sync real demo feeds (images+filters) and serve for screenshots"
	@echo "  make bench           - Run basic performance benchmarks"
	@echo "  make bench-extensive - Run exhaustive multi-CPU benchmarks"
	@echo "  make stress          - Run internal concurrency stress tests"
	@echo "  make load-test   - Run external load generator"
	@echo "  make clean       - Delete binary and data"
	@echo "  make test        - Run tests"
	@echo "  make db-recreate - Clean data and re-seed database"
	@echo ""
	@echo "Rendimiento (ver perf/README.md para el protocolo):"
	@echo "  make perf-seed SEED_N=100000  - Dataset con entradas + indice FTS5"
	@echo "  make test-load                - Carga 10/50/100/200 + RSS del proceso"
	@echo "  make perf-report              - Agrega bench + carga + RSS"
	@echo "  make perf                     - PUERTA PRINCIPAL: seed + bench + carga + informe"
	@echo "  make perf-gate                - Falla ante regresion vs perf/load-baseline.json"

all: install build

# Install dependencies
install:
	@echo "Installing frontend dependencies..."
	cd internal/api/web && npm install
	@echo "Installing backend dependencies..."
	go mod download

# Build only the frontend
build-ui:
	@echo "Building Svelte frontend..."
	cd internal/api/web && npm run build

# Build only the Go binary
build-go:
	@echo "Building Go backend..."
	go build -o $(BINARY_NAME) main.go

# Build both (Embeds UI in binary)
build: build-ui build-go

# Build and run the server immediately
run: build
	@echo "Starting Gorrson on http://localhost:$(PORT)"
	./$(BINARY_NAME) serve --port $(PORT) --data-dir $(DATA_DIR) $(if $(filter true,$(SILENT)),--silent,)

# Run development mode (Frontend)
dev-ui:
	cd internal/api/web && npm run dev

# Run development mode (Backend - needs existing UI build in web/dist)
dev-go:
	go run main.go serve --port $(PORT) --data-dir $(DATA_DIR) $(if $(filter true,$(SILENT)),--silent,)

# Run both in parallel
dev:
	@echo "Starting Gorrson & Vite in parallel..."
	@trap 'kill 0' INT; make dev-go & make dev-ui & wait

# Clean binaries and data
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf internal/api/web/dist
	rm -rf $(DATA_DIR)/*

# Tests & Benchmarks
test: build-ui
	GORRSON_TEST=true go test -v ./internal/...

bench:
	@echo "Running basic benchmarks..."
	go test -bench=. ./internal/db -benchmem

bench-extensive:
	@echo "Running extensive multi-CPU benchmarks (this may take a minute)..."
	go test -bench=. ./internal/db -benchmem -cpu=1,2,4 -benchtime=5s

# Referencia para el gate local de allocs/B/op. Regenerar con 'make bench-baseline'
# solo si el cambio es intencionado, y revisar el diff.
bench-baseline:
	@mkdir -p bench
	GORRSON_TEST=true go test -run=^$$ -bench=. -benchmem -benchtime=200ms -count=6 \
		./internal/... > bench/baseline.txt
	@echo "bench-baseline: referencia en bench/baseline.txt"

stress:
	@echo "Running internal stress tests..."
	go test -v -run TestStress ./internal/api

load-test:
	@echo "Running external load generator (Requires server running at :7077)..."
	go run scripts/stress/main.go

register: build
	./$(BINARY_NAME) register --user $(USER_NAME) --pass $(USER_PASS) --data-dir $(DATA_DIR)

# Reset DB and Seed
reset-db:
	@echo "Resetting and seeding database for user 'user'..."
	@mkdir -p $(DATA_DIR)
	go run scripts/seed/main.go

# Demo para capturas: siembra usuario y feeds en $(DEMO_DIR), añade filtros de
# ejemplo, los sincroniza de verdad (entradas reales con sus imágenes) y
# levanta el servidor. No toca $(DATA_DIR) ni $(PERF_DIR). La UI va embebida,
# así que necesita el frontend construido (build-ui) antes de compilar el
# binario. Requiere internet.
demo: build
	@echo "Sincronizando demo en $(DEMO_DIR) (feeds reales + filtros, puede tardar ~1 min)..."
	@mkdir -p $(DEMO_DIR)
	go run ./scripts/seed -data-dir $(DEMO_DIR) -user $(DEMO_USER) -pass $(DEMO_PASS) -sync -demo-feeds -filters
	@echo ""
	@echo "  gorrson demo en http://localhost:$(DEMO_PORT)"
	@echo "  Usuario: $(DEMO_USER) / $(DEMO_PASS)"
	@echo ""
	GORRSON_JWT_SECRET="$(DEMO_SECRET)" ./$(BINARY_NAME) serve --port $(DEMO_PORT) --data-dir $(DEMO_DIR)

db-recreate: clean reset-db

# ---------------------------------------------------------------------------
# Rendimiento — ver perf/README.md
# ---------------------------------------------------------------------------

# Seed de rendimiento: 42 feeds (los mismos que el seed normal) + N entradas con
# su índice FTS5. A diferencia de `reset-db`, aquí sí se generan entradas: las
# rutas /news y /news?search= no miden nada contra una tabla vacía.
#
#   make perf-seed                    # 10.000 entradas
#   make perf-seed SEED_N=1000000     # 1M (~2m 16s, ~1,1 GiB)
perf-seed:
	@mkdir -p $(PERF_DIR)
	go run ./scripts/seed -data-dir $(PERF_DIR) -n $(SEED_N)

# Micro-benchmarks a fichero: entrada del informe.
BENCH_OUT?=perf/bench.txt
bench-file: build-go
	@mkdir -p perf
	GORRSON_TEST=true go test -run=^$$ -bench=. -benchmem -benchtime=200ms -count=6 \
		./internal/... | tee $(BENCH_OUT)

# Carga contra el proceso real. El generador es un main aparte
# (scripts/perf/loadgen.go) y lee el RSS por /proc/<pid>/status.
#
#   make test-load                    # 10/50/100/200, dataset de 10k
#   make test-load SEED_N=100000      # primero hay que re-sembrar
PERF_OUT?=perf/load-$(shell date +%Y%m%d-%H%M%S).json
test-load: build-go
	@mkdir -p perf
	go run scripts/perf/loadgen.go \
		-binary ./$(BINARY_NAME) \
		-data-dir $(PERF_DIR) \
		-port $(PERF_PORT) \
		-profiles $(PERF_PROFILES) \
		-levels $(PERF_LEVELS) \
		-duration $(PERF_DURATION) \
		-warmup $(PERF_WARMUP) \
		-out $(PERF_OUT)

# Informe agregado. NO ejecuta las mediciones: use `make perf` para eso.
PERF_REPORT_BASE?=perf/report-$(shell date +%Y%m%d-%H%M%S)
BENCH_FILE?=perf/bench.txt
LOAD_JSON?=$(shell ls -t perf/load-*.json 2>/dev/null | head -1)

perf-report:
	@mkdir -p perf
	sh scripts/perf/report.sh "$(BENCH_FILE)" "$(LOAD_JSON)" "$(PERF_REPORT_BASE)"

# Referencia local para `make perf-gate`.
perf-baseline:
	@mkdir -p perf
	@if [ -z "$(LOAD_JSON)" ] || [ ! -r "$(LOAD_JSON)" ]; then \
		echo "perf-baseline: no hay JSON de carga. Ejecuta primero: make test-load"; exit 1; \
	fi
	@cp "$(LOAD_JSON)" perf/load-baseline.json
	@echo "perf-baseline: referencia en perf/load-baseline.json"

# Gate local de rendimiento. NO va en CI: RPS, p95 y RSS dependen de la CPU y de
# la carga de la máquina. Ver scripts/perf_gate.sh para los umbrales.
PERF_RSS_MAX_MIB?=
PERF_P95_MAX_MS?=
perf-gate:
	@sh scripts/perf_gate.sh "$(LOAD_JSON)" "$(PERF_RSS_MAX_MIB)" "$(PERF_P95_MAX_MS)"

# PUERTA PRINCIPAL de rendimiento.
perf: perf-seed bench-file test-load perf-report
	@echo "✔ make perf: informe en perf/report-*.txt y perf/report-*.md"

perf-clean:
	rm -f perf/bench.txt perf/load-[0-9]*.json
	rm -f perf/report-*.txt perf/report-*.md
	@echo "perf-clean: artefactos de rendimiento borrados (baseline conservado)"
