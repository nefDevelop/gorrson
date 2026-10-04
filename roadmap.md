# Gorrson Roadmap & Memory Management

## Current Optimizations (Implemented)
<<<<<<< HEAD
- **Índices de listado (2026-10-03)**: cinco índices nuevos para que `ListEntries` y `ListFeeds` dejen de recorrer la tabla entera. `ORDER BY e.is_priority DESC, e.published_at DESC` no podía usar `idx_entries_published (published_at DESC)` porque `is_priority` encabeza el orden, así que cada petición hacía `SCAN entries` + `USE TEMP B-TREE FOR ORDER BY`: leer 1M de filas y ordenarlas en memoria para devolver 50. Con los índices, `/news` pasa de 2 a 1.991 req/s con 1M de entradas y **deja de degradarse con el tamaño** (antes caía 127 → 14 → 2 req/s de 10k a 1M). `/feeds` mejora 153× al cubrir el conteo de no leídos. Coste: escrituras −21 %, disco +37 %. Los planes de ejecución están fijados por `TestQueryPlansUseIndexes` para que un `ORDER BY` futuro no rompa la cobertura en silencio. Detalle y mediciones en [`bench.md`](bench.md).
=======
>>>>>>> origin/main
- **Scraping Concurrency Limit**: Global semaphore restricts high-memory HTML parsing to a maximum of 5 simultaneous tasks.
- **Sync Concurrency Limit**: Global semaphore restricts concurrent user database synchronization to a maximum of 10 users.
- **Initial Sync Jitter**: Random delay (0-120s) added to background workers at startup to prevent resource spikes (Thundering Herd effect).
- **DOM Sharing & Streaming**: Refactored `ScrapeFullContent` to use `io.Pipe` for streaming to Readability and unified DOM tree processing with `goquery` and `htmlquery` without redundant string conversions.

<<<<<<< HEAD
## Planned Database Tuning

Detalle y mediciones en [`docs/AUDIT.md`](docs/AUDIT.md).

1. **`ORDER BY rank` en la búsqueda**: con un término común (100 % de las
   coincidencias) FTS5 tiene que puntuar y ordenar toda la base antes del
   `LIMIT 50`: ~100 req/s a 100.000 entradas, varios segundos a 1.000.000. Con
   término específico es plano (~1.700 req/s), así que el índice FTS5 no es el
   problema. Arreglarlo exige cambiar la semántica de la búsqueda, no un índice:
   es una decisión de producto.
2. **`/stats` a 1M de entradas** marca el suelo en ~100 ms por los dos `COUNT(*)`,
   que son inherentemente O(n). Resolverlo exige contadores desnormalizados o
   caché; como es una pantalla de ajustes que se abre una vez, no compensa.
3. **`/feeds` sigue siendo lineal** aunque ya no escanea la tabla: recorre las
   entradas no leídas de cada feed. Desnormalizar el contador lo haría O(feeds),
   pero con el índice que se añadió la ganancia restante es pequeña.
4. **Auditoría de consultas**: hecha en [`docs/AUDIT.md`](docs/AUDIT.md) §1.
   De 20 consultas revisadas, solo `fever_id` y los dos `GROUP BY` necesitaban
   cambios; ya están aplicados. El resto usa índices o tablas cuyo tamaño no
   depende de las entradas.

## Security

**Aplicado (2026-10-03), con test de regresión para cada uno:**

- **Secreto JWT fail-closed** (`auth.InitSecret`): el arranque **aborta** si el
  secreto es un valor público conocido (`gorrson-vibrant-rss-secret`,
  `change-me-in-production`) o mide menos de 32 caracteres. Sin secreto
  configurado se genera uno aleatorio con aviso en log. Con esto, el bypass de
  autenticación completo que había en el despliegue por defecto ya no es
  posible.
- **XSS en `/proxy`**: se servía HTML del atacante como `text/html` desde el
  origen de la app (reproducido). Ahora el `Content-Type` es explícito, con
  `nosniff` y CSP `default-src 'none'; sandbox`, y lo que no es imagen se
  rechaza con 415. `DetectContentType` no reconoce SVG ni AVIF, así que la
  cabecera del origen se acepta si es `image/*`; el cuerpo se sirve igual de
  inerte porque manda el `Content-Type` que fijamos.
- **XSS reflejada en `/proxy/page`**: `err.Error()` se escapa con
  `html.EscapeString`.
- **XSS vía feed en el modo lector**: el documento remoto pasa por
  `sanitizer.CleanForProxy` antes de inyectar nuestro propio script. Además
  `resolveURL` ya no deja pasar `javascript:` y el `type="text/plain"` de los
  scripts **sustituye** al atributo en vez de añadirse: con duplicado, el
  navegador ganaba el primero y el script se ejecutaba.
- **DoS de argon2**: limitador por IP y por usuario en `/login`, y límite propio
  en `/api/fever`, que era lo más caro (verificaba la clave contra todos los
  usuarios, 67 MB de RAM cada uno).
- **Creación de bases de datos sin autenticar**: `POST /login` y `X-API-Key` ya
  no crean ficheros; se abre en modo lectura y, si no existe, 401.
- **Timeouts del servidor**: sin ellos, Slowloris.
- **Importar un backup perdía escrituras**: el store cacheado quedaba abierto
  sobre el inode antiguo. Ahora se cierra y se saca de caché.

**Pendiente:** sanear `summary` y re-sanear tras los filtros `REPLACE` (tocan la
ingesta), validar el término de FTS5, y el endurecimiento miscellaneous de
`docs/AUDIT.md` §V12 (logs, `limit` sin techo, errores tragados, cookie sin
`Secure`).

=======
>>>>>>> origin/main
## Planned Memory Tuning (Punto 5)
Si tras las optimizaciones actuales el consumo de RAM sigue siendo elevado en picos de carga:

1. **Ajuste del Garbage Collector (GOGC)**:
   - Configurar la variable de entorno `GOGC` a un valor menor (ej. `GOGC=50`) para que el recolector sea más agresivo a costa de un ligero aumento de CPU.
   - Utilizar `GOMEMLIMIT` para establecer un límite suave de memoria antes de que el GC sea extremadamente agresivo.

2. **Liberación Manual de Memoria**:
   - Evaluar la llamada a `runtime/debug.FreeOSMemory()` después de ciclos de sincronización masivos o tareas de scraping muy pesadas.
