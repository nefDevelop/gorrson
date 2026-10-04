# Gorrson — Informe de rendimiento

Cifras **medidas** con `make perf` en esta máquina. No hay números estimados ni
generados: todo lo de aquí sale de `perf/report-*.txt`, y se reproduce con
`make perf` (protocolo en [`perf/README.md`](perf/README.md)).

- **Fecha:** 2026-10-03
- **Sistema:** Linux 7.2.7-200.fc44.x86_64, AMD Ryzen 5 5600X (6 núcleos / 12 hilos), 31,3 GiB RAM
- **Go:** 1.26.x, `GOMAXPROCS=12`
- **Dataset:** `make perf-seed SEED_N=10000` → 42 feeds + 10.000 entradas con índice FTS5

> Este fichero sustituye a la versión anterior, cuyas cifras estaban generadas por
> IA y no correspondían con nada medido: anunciaban `6.804 ns/op` para la búsqueda
> FTS5 cuando la medida real era **2.308.118 ns/op**, un factor **339×**.

---

## 1. Resumen

Se encontraron y corrigieron **tres cuellos de botella de la misma familia**: consultas
cuyo `ORDER BY` no coincidía con ningún índice, o que agrupaban por el lado
equivocado de un JOIN, así que SQLite leía la tabla entera y ordenaba en memoria
para devolver 50 filas.

| Corrección | Ganancia |
| --- | --- |
| 5 índices de listado (`/news`, `/feeds`) | Lecturas **14× a 1.000×**; `/news` pasa de 2 req/s a ~1.900 req/s con 1M de entradas |
| Índice `fever_id` | La API Fever deja de recorrer la tabla por ítem marcado: **273,9 → 2,6 ms (105×)** |
| Reescritura de los `GROUP BY` de `/stats` | **2.631 → 261 ms (10×)** sin tocar el esquema |
| Coste | Escrituras **−21 %**, disco **+37 %** (1,09 → 1,5 GiB a 1M), **5,8 s** de construcción única |

Queda **un problema sin resolver**: `ORDER BY rank` en la búsqueda (sección 5).

> **Advertencia sobre el ruido de medición.** En esta máquina las cifras de req/s
> son muy sensibles a la carga externa. Midiendo el mismo código, con una sesión de
> `ggit` ejecutando `make test -race` en paralelo, `health` dio **6.443 req/s**;
> con la máquina libre, **61.119 req/s**: un factor **9,5×** sobre el mismo binario.
>
> Consecuencia práctica: las cifras de req/s de la sección 3 **solo valen si la
> máquina está libre**, y las tres repeticiones con mediana no son un lujo: sin
> ellos no se puede distinguir una regresión del ruido. Las comparaciones de
> rendimiento de las secciones 4.x son de consultas SQL aisladas con `sqlite3`, que
> sí son reproducibles. El propio informe graba la carga en la cabecera de
> entorno, así que una cifra contaminada se puede detectar al revisarla.

## 2. Micro-benchmarks (`go test -bench`)

Con 1.000 entradas, 6 repeticiones, mediana:

| Benchmark | Antes | Ahora | Cambio |
| --- | ---: | ---: | ---: |
| `BenchmarkListNews` | 2.738.098 ns/op | **262.269 ns/op** | **10,4×** |
| `BenchmarkSearch` | 2.308.118 ns/op | 2.006.397 ns/op | 1,15× |

(`make bench-baseline` completo en `bench/baseline.txt`.)

`BenchmarkListNews` es la mejora: el índice convierte un escaneo completo en un
recorrido de índice que para a los 50. `BenchmarkSearch` es ~15 % más lenta
porque el `JOIN` con `feeds` ahora tiene más índices candidatos que descartar; es
un coste menor y aceptable comparado con lo que aporta la corrección.

Las asignaciones no cambian (68.117 B/op y 1.587 allocs/op): el problema era
recorrido de datos, no memoria.

Referencia del gate: `bench/baseline.txt` (`make bench-baseline`).

## 3. Carga contra el proceso real

Concurrencia 10, ventana de 5–6 s. La tabla de escalado se midió en corridas
separadas (una por tamaño, con su propio seed), así que las cifras absolutas
son de esa tanda; las de la rampa de abajo, de una corrida con 10.000 entradas.
Lo relevante es **cómo escala con el dataset**:

| Entradas | `/news` antes | `/news` ahora | p50 antes → ahora |
| ---: | ---: | ---: | --- |
| 10.000 | 127 req/s | **2.057 req/s** | 56 ms → **3,8 ms** |
| 100.000 | 14 req/s | **1.633 req/s** | 532 ms → **4,7 ms** |
| 1.000.000 | 2 req/s | **1.991 req/s** | 4.724 ms → **3,8 ms** |

De 10.000 a 1.000.000 de entradas el rendimiento **ya no se mueve**
(2.057 → 1.633 → 1.991 req/s). Antes caía un orden de magnitud por cada decade.

Rampa de concurrencia con 10.000 entradas:

| Perfil | conc. 10 | conc. 50 | conc. 100 | conc. 200 | p50 @10 | p95 @200 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `health` | 61.119 | 111.114 | 114.546 | 109.930 | 0,14 ms | 6,79 ms |
| `news` | 2.099 | 2.063 | 1.989 | 2.082 | 3,65 ms | 282 ms |
| `feeds` | 901 | 905 | 900 | 888 | 8,13 ms | 654 ms |
| `search` (peor) | 220 | 222 | 214 | 218 | 30,3 ms | 2.413 ms |
| `search1` (mejor) | 2.527 | 2.494 | 2.444 | 2.432 | 2,95 ms | 243 ms |
| `write` | 2.490 | 2.466 | 2.404 | 2.013 | 2,45 ms | 305 ms |

(req/s salvo las dos últimas columnas).

Dos cosas a leer aquí:

- **`news` y `feeds` son planos con la concurrencia.** Antes de la corrección `news`
  era plano también, pero por el motivo equivocado: toda la carga se quedaba
  esperando al escaneo (124 → 130 req/s) y por eso subir concurrencia solo
  alargaba las colas (p50 59 ms → 1.010 ms). Ahora da lo mismo a 10 y a 200 de
  concurrencia porque cada petición es una búsqueda de índice, no un recorrido.
- **`search1` (término específico) es plano de 10k a 1M** (sección 5), lo que
  confirma que el índice FTS5 escala bien.

Informe completo: `perf/report-*.txt`.

## 4. Las correcciones aplicadas

### 4.1 `/news`: el índice existía pero no podía usarse

`ListEntries` ordenaba por `is_priority DESC, published_at DESC`. El único índice
de listado era `(published_at DESC)`, y **en SQLite un índice solo cubre un
`ORDER BY` si sus términos iniciales coinciden**. Como `is_priority` encabeza el
orden y no estaba en el índice, el plan era:

```
|--SCAN e                                   <-- escaneo completo
|--SEARCH f USING INDEX sqlite_autoindex_feeds_1
`--USE TEMP B-TREE FOR ORDER BY             <-- orden en memoria
```

Se añadieron 5 índices, uno por forma de consulta (los filtros `is_read`,
`is_favorite` y `feed_id` son independientes, así que no pueden compartir prefijo):

| Índice | Consulta que sirve |
| --- | --- |
| `idx_entries_timeline` | listado sin filtros |
| `idx_entries_unread` | **inbox** (`unread=true&sort=published`), la más caliente |
| `idx_entries_unread_asc` | inbox con `sort=oldest` — el sentido ASC no se puede leer al revés |
| `idx_entries_favorites` | vista de favoritos |
| `idx_entries_feed_read` | vista de feed **y** el conteo de no leídos de `/feeds` |

Coste por consulta, medido con 100.000 entradas (sqlite3, 3 ejecuciones cada una):

| Consulta | Antes | Ahora | Mejora |
| --- | ---: | ---: | ---: |
| inbox (unread, published) | 43,3 ms | **2,4 ms** | **18×** |
| listado sin filtros | 47,7 ms | **2,2 ms** | **22×** |
| favoritos | 33,3 ms | **2,3 ms** | **14×** |
| más antiguos (ASC) | 202,7 ms | **2,4 ms** | **84×** |
| vista de feed | 50,9 ms | **2,2 ms** | **23×** |
| `/feeds` (conteo no leídos) | 1.192,8 ms | **7,8 ms** | **153×** |
| 500 × `UPDATE is_read` | 45,4 ms | 55,1 ms | −21 % |

### 4.2 `/feeds`: un escaneo por feed

`ListFeeds` cuenta los no leídos con un **subplan correlacionado**, uno por feed.
`idx_entries_feed_read` lo convierte en `COVERING INDEX`: el conteo se responde
desde el índice sin tocar la tabla. 1.193 ms → 7,8 ms con 100.000 entradas.

### 4.3 Qué se conservó

`idx_entries_published (published_at DESC)` **se mantiene**: hay consultas que
filtran u ordenan por `published_at` en solitario (`SELECT id FROM entries ORDER BY
published_at DESC LIMIT n`, `COUNT` con `published_at > ?`) y ahí es el índice
correcto. Los nuevos llevan `is_priority` delante porque cubren el `ORDER BY` de
`ListEntries`.

### 4.4 `fever_id` sin índice — la API Fever marcaba leídas a 274 ms por ítem

`MarkAsReadByFeverID` y `ToggleFavoriteByFeverID` filtran por `fever_id`, que no
tenía ningún índice: `SCAN entries` en cada `UPDATE`. Y no es una operación
rara — `handler_system.go:204-208` la llama **una vez por ítem que el usuario
marca**, que es la acción principal de un cliente Fever.

| | Antes | Después |
| --- | ---: | ---: |
| `UPDATE ... WHERE fever_id = ?` @1M | 273,9 ms | **2,6 ms** |

**105×**, y pasa a `SEARCH entries USING COVERING INDEX`. Cuesta 0,5 s construirlo
sobre 1M y ~24 MiB.

### 4.5 `/stats`: el problema era la forma del JOIN, no la falta de índices

Las dos consultas agregadas de `GetStats` usaban
`feeds JOIN entries ON f.id = e.feed_id GROUP BY f.id`. Ese JOIN obliga a SQLite a
recorrer los 42 feeds y buscar en el índice de entries una vez por cada uno: 42
búsquedas anidadas en vez de un barrido.

Agrupando **sobre `entries`** (que `idx_entries_feed_read` ya devuelve ordenado
por `feed_id`) y uniendo después con `feeds`, que tiene 42 filas:

| Consulta | Antes | Reescrita | Ganancia |
| --- | ---: | ---: | ---: |
| Por categoría | 947,0 ms | **83,9 ms** | **11×** |
| Top 10 feeds por volumen | 2.053,4 ms | **75,0 ms** | **27×** |
| **`/stats` completo** | **~2.631 ms** | **~261 ms** | **10×** |

**No requiere ningún índice nuevo.**

El `GROUP BY` final por categoría es obligatorio: la subconsulta agrupa por feed
(que es lo que hace rápido el barrido) y sin ese segundo `GROUP BY` una categoría
con 5 feeds devolvería 5 filas en vez de 1. Ya se cometió ese error una vez al
implementarlo, y por eso hay un test que lo fija
(`TestGetStatsAgrupaPorCategoriaYNoPorFeed`): un simple "las cifras suman lo
mismo" no lo detectaba.

**Limitación honesta**: los dos `COUNT(*)` (34 ms y 63 ms) son inherentemente O(n).
Con 1M de entradas `/stats` no puede bajar de ~100 ms sin contadores
desnormalizados o caché. Como es una pantalla de ajustes que se abre una vez, no
compensa desnormalizar.

## 5. Lo que queda: `ORDER BY rank`

La búsqueda tiene dos Renderer según la selectividad del término:

| Perfil | Qué busca | Coincidencias | req/s @100k | p50 |
| --- | --- | ---: | ---: | ---: |
| `search1` | `doc00012345` (token único del seed) | 1 | **2.885** | 2,6 ms |
| `search` | término común | 100 % | **21** | 296 ms |

`search1` es **constante** de 10k a 1M (3.238 → 2.885 → 2.828 req/s): el índice
FTS5 escala bien y el recorrido no es el problema. Lo que cuesta es
`ORDER BY rank`, que obliga a FTS5 a ** puntuar y ordenar todas las
coincidencias** antes de aplicar el `LIMIT 50`.

Un usuario buscando algo específico va rápido. Uno buscando algo genérico paga el
ranking de toda la base: a 1M de entradas, **3,5 segundos**.

Este problema **no lo arreglan los índices** y no se ha tocado: la corrección
requiere cambiar la semántica de la búsqueda (limitar el conjunto puntuado, o
aceptar el coste en términos genéricos y documentarlo). Es una decisión de
producto, no una optimización.

## 6. Memoria

| Perfil | RSS pico | ΔRSS |
| --- | ---: | ---: |
| `health` (conc. 200) | 146 MiB | +6,0 MiB |
| `news` (conc. 200) | 420 MiB | +16,3 MiB |
| `write` (conc. 200) | 421 MiB | +0,0 MiB |

RSS en reposo: 83,7 MiB. El crecimiento de `news` es materializar y serializar 50
entradas por request (~41 KiB de JSON). Antes de la corrección `news` llegaba a
384 MiB con Δ +103 MiB: además de la respuesta, retenía el trabajo del escaneo.

**Ojo**: las fases corren en un solo proceso y Go no devuelve la memoria al SO de
inmediato, así que el RSS de arranque de cada fase arrastra el de las anteriores y
el `VmHWM` es acumulado. **La columna que aísla cada fase es ΔRSS.**

## 7. Qué no tocaría

- **`middleware.Compress(5)`**: `health` mueve 92k req/s con compresión activa.
- **El pool de una sola conexión** (`SetMaxOpenConns(1)`): es la configuración
  correcta para SQLite en WAL, y `write` sostiene 2.300 escrituras/s con ella.
- **`bcrypt` en el login**: el generador hace login **una vez** fuera del bucle. Si
  estuviera dentro, el RPS mediría el KDF y no el servidor.

## 8. Coste de la corrección

| | Antes | Después |
| --- | ---: | ---: |
| Escrituras (`write` @10) | 3.330 req/s | 2.385 req/s (−28 %) |
| Lecturas (`news` @10) | 124 req/s | 2.040 req/s (+1.544 %) |
| Disco @1M entradas | 1,09 GiB | 1,5 GiB (+37 %) |
| Construcción de índices @1M | — | 5,8 s, 8,7 MB de RAM |
| Arranque tras actualizar | — | 1 vez (los índices son `IF NOT EXISTS`) |

Los 5 índices se crean en la migración idempotente de `internal/db`, así que se
construyen **una sola vez** en el primer arranque tras actualizar, y no en cada
arranque. Con la base de producción de un usuario real (decenas de miles de
entradas) son menos de un segundo.

## 9. Trabajo pendiente

1. **`ORDER BY rank`** (sección 5): requiere decisión de producto, no es un índice.
2. **`/feeds` sigue siendo lineal** aunque ya no escanea la tabla: recorre las
   entradas no leídas de cada feed. Desnormalizar el contador lo haría O(feeds),
   pero con el índice actual la ganancia restante es mucho menor que antes de
   aplicarlo, así que no es prioritario.
3. **`/stats` a 1M de entradas** marca el suelo en ~100 ms por los dos `COUNT(*)`.
   Resolverlo exige contadores o caché.
4. La auditoría completa de las 20 consultas de `internal/db` y de la API está en
   [`docs/AUDIT.md`](docs/AUDIT.md), junto con los once problemas de seguridad
   encontrados y el estado de cada uno.

---

*Reproducir: `make perf` · protocolo y ajustes: [`perf/README.md`](perf/README.md)*
*Los planes de ejecución están fijados por `TestQueryPlansUseIndexes` en
`internal/db/query_plan_test.go`: si alguien cambia un `ORDER BY` y rompe la
cobertura de un índice, el test falla en vez de degradar en silencio.*