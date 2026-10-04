# Rendimiento de gorrson

Instrumentación de rendimiento: micro-benchmarks, prueba de carga contra el
proceso real y consumo de RSS.

## Comandos

```bash
make perf-seed SEED_N=10000     # dataset: 42 feeds + N entradas con índice FTS5
make bench                       # micro-benchmarks (existente)
make bench-file                  # micro-benchmarks a perf/bench.txt
make test-load                   # carga: 10/50/100/200 de concurrencia
make perf-report                 # agrega bench + carga + RSS -> perf/report-*.txt|.md
make perf                        # PUERTA PRINCIPAL: seed + bench + carga + informe
make perf-gate                   # falla ante regresión vs perf/load-baseline.json
make perf-baseline               # regenera la referencia del gate
```

Ajustes:

```bash
make perf SEED_N=100000                       # dataset grande
make perf-seed SEED_N=1000000                 # 1M: ~2m 16s, ~1,1 GiB
make test-load PERF_LEVELS=50,100             # solo esos niveles
make test-load PERF_DURATION=30s              # ventanas más largas
make test-load PERF_PROFILES=news,write       # un subconjunto de perfiles
make perf-gate PERF_RSS_MAX_MIB=300           # techo absoluto de RSS
make perf-gate PERF_P95_MAX_MS=200            # techo absoluto de p95
```

## Los cinco perfiles de carga

Cada perfil aísla una capa distinta. Si se mezclan, los números no dicen nada:

| Perfil | Petición | Qué aísla |
| --- | --- | --- |
| `health` | `GET /health` | Suelo de HTTP puro: router, `Compress(5)`, cabeceras de seguridad. No toca la DB. |
| `news` | `GET /news?limit=50&unread=true&sort=published` | La lectura más caliente: `ListEntries` con JOIN a `feeds` y `ORDER BY`. |
| `search` | `GET /news?search=<término común>` | **Peor caso** de búsqueda: el término está en el 100 % de las entradas, así que `ORDER BY rank` puntúa y ordena todas las coincidencias. |
| `search1` | `GET /news?search=docNNNNNNNN` | **Mejor caso**: el token único que el seed escribe en cada entrada, una sola coincidencia. Comparado con `search`, acota cuánto del coste es el ranking y cuánto el resto. |
| `write` | `PUT /news/{id}/read` | Escritura sobre SQLite en WAL con `SetMaxOpenConns(1)`: mide la serialización de las escrituras. |

**El login va fuera del bucle de carga.** El generador hace un único `POST /login`
y reutiliza el `Bearer` token. Con `bcrypt` dentro del bucle, el RPS mediría el
KDF en vez del servidor.

## Dataset: qué escala y qué no

`make perf-seed SEED_N=N` escala **`entries`**, que es lo que miden `/news` y
`/news?search=`. El índice FTS5 se puebla solo por el trigger `entries_ai`, así
que la búsqueda funciona igual que con el sync real.

| Tabla | Filas | Por qué |
| --- | --- | --- |
| `users` | 1 | Un usuario es un usuario. |
| `feeds` | 42 | Los mismos 42 fuentes del seed normal: se usan como filtro (`category`, `feed_id`), no como volumen. |
| **`entries`** | **`SEED_N`** | **Lo que se mide.** Alimenta `/news` (escaneo + orden) y `/news?search=` (FTS5). |
| `filters` | 0 | Se siembran a mano si hacen falta. |

Las entradas llevan un token único `docNNNNNNNN` además del vocabulario común.
Sin él no se puede medir el mejor caso de búsqueda: todos los términos devolverían
el 100 % de las filas y `search` y `search1` serían el mismo número.

Costes medidos en esta máquina:

| `SEED_N` | Tiempo de seed | Tamaño de DB |
| ---: | ---: | ---: |
| 10.000 | 1,2 s | 11,0 MiB |
| 100.000 | 13,9 s | 108,5 MiB |
| 1.000.000 | 135,9 s | 1.086,7 MiB |

El directorio del dataset es `./data/perf/`, siempre aparte de `./data/` de
desarrollo. Está en `.gitignore`: a 1M son 1,1 GiB.

## Protocolo de medición

1. **Un proyecto a la vez**, nada más corriendo. La máquina tiene 12 hilos y es
   de escritorio: una sesión de navegador o un `go build` en segundo plano
   contaminan la latencia.
2. **Secuencial, nunca en paralelo.**
3. **Tres repeticiones, mediana.** Para comparar dos versiones, `-duration 30s`
   por fase como mínimo.
4. **El warmup no se contabiliza** (`PERF_WARMUP`).
5. **La cabecera de entorno va siempre en el informe**: CPU, núcleos, RAM, kernel,
   versión de Go, `GOMAXPROCS` y tamaño del dataset.

## RSS: cómo se mide

El generador lee `/proc/<pid>/status` cada 50 ms: `VmRSS` (actual) y `VmHWM`
(pico). Se mide **antes** de la carga para tener el RSS en reposo, sin el cual
"420 MiB" no dice nada.

**Cuidado al interpretarlo**: las fases corren en un solo proceso, en orden, y Go
no devuelve la memoria al SO de inmediato. El RSS de arranque de una fase arrastra
el de las anteriores y el `VmHWM` es acumulado de todo el proceso. **La columna que
aísla cada fase es ΔRSS.**

Un Δ positivo y sostenido indica **memoria retenida**, no fuga confirmada. Antes de
acusar: `make test-race`, y repetir tras el `LimpiarRateLimiter()` equivalente.

## Gates

- `make bench-baseline` / `bench/baseline.txt` — referencia de `allocs/op` y
  `B/op`. **Independiente de la máquina**, por eso es el único gate que tendría
  sentido en CI.
- `make perf-gate` — RSS pico +15 %, p95 +25 %, RPS −10 % contra
  `perf/load-baseline.json`, más techos absolutos opcionales. **Solo local**: RPS,
  p95 y RSS dependen de la CPU y de la carga de la máquina.

Comparación fase a fase por `(perfil, concurrencia)`: son las celdas que deben
seguir siendo comparables entre corridas.

## Ficheros

```
scripts/perf/loadgen.go     # generador + lector de RSS (main aparte, //go:build ignore)
scripts/perf/report.sh      # agregador del informe
scripts/perf_gate.sh        # gate de carga
perf/                       # informes y referencias (generado)
bench/baseline.txt          # referencia de micro-benchmarks (versionada)
```

`loadgen.go` lleva `//go:build ignore` a propósito: así queda fuera de `./...` y no
aparece en `make test` ni en `go vet ./...`. No toca ninguna ruta de código de
producción.

## Resultados

Ver [`../bench.md`](../bench.md) para el informe con las cifras medidas y el
diagnóstico de los dos cuellos de botella (`/news` no usa su índice,
`ORDER BY rank` no escala).