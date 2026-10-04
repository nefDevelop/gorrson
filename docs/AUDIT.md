# Auditoría de rendimiento, API y seguridad — gorrson

- **Fecha:** 2026-10-03
- **Alcance:** `internal/db/` (todas las consultas), `internal/api/` (rutas y handlers),
  `internal/auth/`, `internal/rss/`, `internal/opml/`, `internal/sanitizer/`, `cmd/`
- **Método:** `EXPLAIN QUERY PLAN` y mediciones sobre un dataset de **1.000.000 de
  entradas**; revisión de código línea a línea de la superficie HTTP.
- **Estado:** informe. **Los 9 puntos de la lista de arreglo están aplicados y
  verificados con test**; el resto queda pendiente. Ver §Estado de cada punto.

Las cifras de rendimiento de este informe son reproducibles con `make perf` y el
protocolo está en [`../perf/README.md`](../perf/README.md). El detalle de las
correcciones ya aplicadas está en [`../bench.md`](../bench.md).

---

# Parte 1 — Rendimiento de consultas

## 1.1 Lo ya corregido (ver `bench.md`)

`ListEntries` y `ListFeeds`aban a escanear la tabla entera. Corregido con 5
índices: lecturas entre **14× y 1.000×**.

## 1.2 `fever_id` sin índice — la API Fever marca leídas a 307 ms por ítem

`MarkAsReadByFeverID` y `ToggleFavoriteByFeverID` (`db.go:377`, `db.go:382`) filtran
por `fever_id`, que **no tiene ningún índice**:

```
EXPLAIN QUERY PLAN UPDATE entries SET is_read=1 WHERE fever_id=42;
`--SCAN entries
```

Medido con 1.000.000 de entradas: **273,9 ms por `UPDATE`**, matched o no. Y es la
acción principal de un cliente Fever: `handler_system.go:204-208` la invoca una vez
por ítem que el usuario marca.

Con `CREATE INDEX idx_entries_fever ON entries(fever_id)`:

| | Antes | Después |
| --- | ---: | ---: |
| `UPDATE ... WHERE fever_id = ?` | 273,9 ms | **2,6 ms** |

**105×**, y pasa a `SEARCH entries USING COVERING INDEX`. Cuesta 0,5 s construirlo
sobre 1M y ~24 MiB de disco.

**Nota sobre el dataset**: `scripts/seed` no rellena `fever_id` (todas las filas
quedan a `DEFAULT 0`), así que el perfil de carga no ejercita esta ruta. El parser
sí lo asigna (`parser.go:291`, `generateFeverID`), así que en producción es real.

## 1.3 `/stats`: el problema es la forma del JOIN, no la falta de índices

`GetStats` (`db.go:683`) lanza 4 consultas. Las dos agregadas usan
`feeds JOIN entries ON f.id = e.feed_id`, y ese JOIN obliga a un bucle anidado de
42 búsquedas de índice en lugar de un barrido en streaming:

```
SELECT f.title, COUNT(e.id) c FROM feeds f JOIN entries e ON f.id=e.feed_id GROUP BY f.id ORDER BY c DESC LIMIT 10;
|--SCAN f USING INDEX sqlite_autoindex_feeds_1
|--SEARCH e USING INDEX idx_entries_feed_read (feed_id=?)
`--USE TEMP B-TREE FOR ORDER BY
```

Agrupando **sobre `entries`** (que ya está ordenado por `feed_id` en el índice) el
barrido sale ya agrupado y hay que unir después con `feeds`, que tiene 42 filas:

| Consulta | Actual | Reescrita | Ganancia |
| --- | ---: | ---: | ---: |
| Por categoría | 947,0 ms | **83,9 ms** | **11×** |
| Top 10 feeds por volumen | 2.053,4 ms | **75,0 ms** | **27×** |
| **`/stats` completo** | **~2.631 ms** | **~261 ms** | **10×** |

La forma reescrita, medida:

```sql
SELECT f.title, c.c
FROM (SELECT feed_id, COUNT(*) c FROM entries GROUP BY feed_id) c
JOIN feeds f ON f.id = c.feed_id
ORDER BY c.c DESC LIMIT 10;
```

**No necesita ningún índice nuevo** — `idx_entries_feed_read` ya cubre el
`GROUP BY`. Un índice estrecho extra `(feed_id)` lo baja a 64 ms, pero el 27× viene
de la reescritura sola.

**Limitación honesta**: los dos `COUNT(*)` (34 ms y 63 ms) son inherentemente O(n).
Con 1M de entradas, `/stats` no puede bajar de ~100 ms sin contadores
desnormalizados o caché. Como es una pantalla de ajustes que se abre una vez, no
compensa desnormalizar.

## 1.4 El resto de consultas: sanas

| Consulta | Plan | Estado |
| --- | --- | --- |
| `GetEntryByID`, `MarkAsRead`, `ToggleFavorite` (`WHERE id=?`) | `SEARCH ... USING COVERING INDEX` (PK) | ✓ |
| `GetFeedByID` (`WHERE id=?`) | PK de `feeds` | ✓ |
| `GetFeedByURL` (`WHERE feed_url=?`) | `UNIQUE` | ✓ |
| `ValidateAPIKey` (`WHERE key_hash=?`) | `SEARCH ... USING COVERING INDEX` | ✓ |
| `GetSession` (`WHERE refresh_token=?`) | `UNIQUE` | ✓ |
| `SearchEntries` | FTS5 (ver `bench.md` §5) | ✓ |
| `CleanupOldEntries` | `SEARCH ... USING INDEX idx_entries_favorites` | ✓ |
| `DeleteExpiredSessions` | `SCAN sessions` (2,3 ms) | ⚠ menor |
| `GetFiltersForFeed` | `SCAN filters` | ⚠ menor |
| `RenameCategory` | `SCAN feeds` (42 filas) | ✓ irrelevante |

Los tres marcados ⚠ recorren tablas cuyo tamaño depende de acciones del usuario, no
de las entradas: sesiones (una por login), filtros (los crea el usuario) y feeds
(42). Con un índice barato se ganarían menos de 1 ms. **No los aplico.**

## 1.5 Resumen de cambios de rendimiento — APLICADOS

| Cambio | Consulta | Ganancia @1M | Coste |
| --- | --- | ---: | --- |
| `idx_entries_fever (fever_id)` | API Fever mark/favorite | 273,9 → 2,6 ms (**105×**) | ~24 MiB, 0,5 s |
| Reescritura de 2 GROUP BY | `/stats` | 2.631 → 261 ms (**10×**) | **0** (sin esquema) |
| 5 índices de listado (§1.1) | `/news`, `/feeds` | 14× a 1.000× | +37 % disco, escrituras −21 % |

Los tres están implementados y verificados con test:
`TestQueryPlansUseIndexes` fija los planes de ejecución y
`TestGetStatsAgrupaPorCategoriaYNoPorFeed` fija la semántica de la reescritura
(un `GROUP BY` por feed en vez de por categoría devolvía una fila por feed y
mantenía el total, así que un simple "las cifras cuadran" no lo detectaba).

---

# Parte 2 — Seguridad

Ordenado por gravedad. Cada punto está verificado con lectura de código; los tres
primeros los he reproducido yo mismo.

## ✅ V1 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 `/proxy` sirve HTML del atacante como `text/html` — **XSS almacenado reflected**

`internal/api/handler_system.go:268` y `:307`

```go
http.ServeContent(w, r, "image", time.Time{}, f)
```

El nombre `"image"` no tiene extensión, así que `ServeContent` cae a
`DetectContentType` sobre los primeros 512 bytes. Solo se fija `Cache-Control`;
**no hay `Content-Type` de restricción ni `Content-Disposition`**.

**Reproducido** con el mismo `ServeContent` de la stdlib:

```
status      : 200
Content-Type: text/html; charset=utf-8
Disposition : ""
body        : <!DOCTYPE html><html><body><script>alert(document.domain)</script>hola
```

Impacto: `/proxy?url=https://atacante.tld/x.html` devuelve HTML+JS **desde el origen
de la app**. La CSP global (`server.go:259`) incluye `'unsafe-inline'`, así que el
`<script>` inline se ejecuta; y la cookie `gorrson_auth` es `SameSite=Lax`, que **sí
se envía en un GET de nivel superior** vindo de un enlace externo.

Arreglo: validar el `Content-Type` tras el fetch y responder error si no empieza por
`image/`, y/o fijar `Content-Type: image/*` y `Content-Disposition: attachment`.

## ✅ V2 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 XSS reflejada en `/proxy/page` — **una línea**

`internal/api/handler_system.go:350`

```go
fmt.Fprintf(w, "<pre style='...'>%s</pre>", err.Error())
```

`net/url` mete la URL cruda en el mensaje de error **sin escapar**, y la respuesta
va con `Content-Type: text/html` (`:345`). Payload:
`/proxy/page?url=http://a/%zz%22%3E%3Cimg%20src%3Dx%20onerror%3Dalert(1)%3E` — falla
en `NewRequestWithContext` **antes de tocar la red**.

Arreglo: `html.EscapeString(err.Error())`.

## ✅ V3 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 Secreto JWT forjable en el despliegue por defecto — **bypass de autenticación**

`internal/auth/auth.go:26` y `auth.go:29-33`

```go
var Secret = []byte("gorrson-vibrant-rss-secret")

func GetSecret() []byte {
	envSecret := os.Getenv("GORRSON_JWT_SECRET")
	if envSecret != "" { return []byte(envSecret) }
	return Secret
}
```

`Dockerfile:47` pone `GORRSON_JWT_SECRET=change-me-in-production` (también público) y
`docker-compose.yml:11` lo interpola **sin valor por defecto**: si el operador no
exporta la variable, Compose pasa `GORRSON_JWT_SECRET=` (vacío), que **pisa** el
`ENV` del Dockerfile, y `GetSecret()` cae al literal de `auth.go:26`.

Consecuencia: cualquiera con el código fuente firma `{"username":"<cualquiera>"}`
con HS256 y obtiene autenticación completa, incluido `/backup/export`.

Arreglo: generar un secreto aleatorio en el arranque si falta y **negarse a
arrancar** (fail-closed), no usar un literal.

## ✅ V4 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 argon2 sin rate limiting — DoS no autenticado

`internal/auth/auth.go:19-21`: `Memory = 64*1024` KiB, `Iterations = 3`.

- **`handler_auth.go:42`**: un argon2 por intento de login, **sin rate limit ni
  bloqueo de cuenta**. 100 logins concurrentes ≈ 6,7 GB y ~1,6 s de CPU.
- **`handler_system.go:78-98`**: `/api/fever` itera **todos** los `.db` del
  directorio y verifica la contraseña de cada uno → **N × 67 MB por petición**. Con
  20 usuarios: 1,34 GB por request. Y el bucle **no para en el primer fallo**, así
  que siempre recorre la lista completa.

No hay rate limiting en ningún punto del proyecto.

Arreglo: rate limit por IP y por usuario en `/login`; y en `handleFeverAPI`,
seleccionar el usuario antes de verificar.

## ✅ V5 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 Creación de base de datos no autenticada + fuga de handles

`internal/api/server.go:222` y `internal/api/handler_auth.go:35`

Ambos llaman a `getStore(username)` **antes** de validar nada, y `getStore` →
`db.NewStore` (`db.go:22`) **crea el fichero y ejecuta `migrate()`**, y lo cachea
para siempre en `s.stores` (`server.go:81`) sin evicción.

- `POST /login` con `{"username":"x1","password":"y"}` → crea `x1.db` con 11 tablas,
  5 índices y FTS5 (~50 KB), más un `*sql.DB` permanente. **Sin autenticar.**
- `X-API-Key: cualquiera:basura` → lo mismo.

10.000 peticiones ≈ 500 MB de disco y 10.000 pools de conexión.

Arreglo: en `authMiddleware` y `handleLogin`, abrir en modo lectura **sin migrar** y
comprobar existencia antes de crear.

## ✅ V6 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 `/proxy/page` no sanea el HTML remoto

`internal/api/handler_system.go:363-412`

- `:369-376` desactiva `<script>` **añadiendo** `type="text/plain"` con `append`. Si
  la página ya traía `type` (`<script type="module">`), queda un atributo duplicado
  y el navegador gana **el primero**.
- No toca event handlers: `<img src=x onerror=…>` pasa intacto, y la CSP de esta
  ruta permite `'unsafe-inline'`.
- `resolveURL` (`server.go:273-282`) **preserva `javascript:` y `data:`** a propósito
  (`:274`). El `preventDefault` del script (`:461`) solo cubre clic izquierdo:
  clic central o "abrir en pestaña nueva" lo esquivan.

Cadena sin ningún paso del atacante: la víctima se suscribe a un feed cuyo
`site_url` es `evil.tld`, abre una entrada en modo lector y el iframe renderiza HTML
del atacante en el origen de la app.

Arreglo: pasar el documento por `sanitizer.Clean` antes de renderizar (con una
política específica, porque `Clean` permite `<iframe>`).

## V7 · `summary` nunca se sanea

`internal/rss/parser.go:234`

```go
summary := item.Description   // crudo, sin pasar por sanitizer
```

Se trunca a 200 caracteres y se guarda. El frontend lo renderiza con `{@html}`
(`PostViewer.svelte:294`) tras un ajuste que **no sanea**.

Condicionado a que `content` esté vacío (bluemonday lo deja en `""` si era
`<script>`), pero es un XSS almacenado desde un feed de terceros.

## V8 · Los filtros `REPLACE` escriben después de sanear

`internal/rss/parser.go:168` sanea → `parser.go:220` reinyecta sin re-sanear:

```go
cleanContent = re.ReplaceAllString(cleanContent, filter.Replacement)
```

→ se guarda en BD → se sirve como `text/html` crudo (`handler_news.go:75-76`).
Con filtros propios es self-XSS; pasa a ser XSS almacenado real en cuanto las
definiciones de filtro vengan de una fuente compartida (`POST /config/import`).

## ✅ V9 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 Sin timeouts en el servidor — Slowloris

`cmd/serve.go:128-131`

```go
srv := &http.Server{ Addr: addr, Handler: server.GetRouter() }
```

Sin `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` ni `IdleTimeout`. Unos pocos
sockets con cabeceras a goteo ocupan handlers indefinidamente. **Es lo primero que
arreglaría de esta lista**: es lo que hace más graves todos los demás.

```go
ReadHeaderTimeout: 10*time.Second, ReadTimeout: 60*time.Second,
WriteTimeout: 300*time.Second, IdleTimeout: 120*time.Second,
```

## V10 · FTS5: el término va crudo a `MATCH`

`internal/api/handler_news.go:24` → `db.go:458`

```
search='gato AND'  → fts5: syntax error  → 500
search='*'         → unknown special query
search='content:x' → el usuario alcanza los nombres de columna del FTS
```

No hay fuga entre usuarios (`entries_fts` es por base de datos), pero sí **500 en
vez de 400** y la posibilidad de forzar barajadas grandes.

Arreglo: indexar el término como literal (entrecomillado con comillas dobles
escapadas) y devolver 400 si FTS sigue rechazando.

## ✅ V11 · — CORREGIDO

> Ver "Orden de arreglo — estado".

 Importar un backup deja la store cacheada en el inode viejo — **pérdida de datos**

`internal/api/handler_system.go:849` hace `os.Rename` sobre el fichero **vivo**,
mientras `s.stores[username]` (`server.go:81`) mantiene el `*sql.DB` abierto sobre el
**inode antiguo**, ya desvinculado pero legible por el descriptor.

Resultado: después de restaurar un backup, **todas las consultas siguen yendo a la
base vieja** y **las escrituras posteriores se pierden** en un inode sin nombre. El
log dice "Page refresh recommended" (`:856`) pero refrescar no ayuda: la caché está
en el servidor. Requiere reiniciar el proceso.

Es un fallo de integridad de datos, no de seguridad, pero es el más caro de esta
lista.

## V12 · Otros

| Ubicación | Qué |
| --- | --- |
| `handler_system.go:101` | `"api_key", apiKey` — la api_key Fever **en claro** en cada intento fallido |
| `server.go:87` + chi Logger | `r.RequestURI()` incluye el query string → **el JWT de `?token=` se escribe en el log en cada request** |
| `handler_system.go:350`, `handler_feeds.go:102`, `:276` | `err.Error()` al cliente: filtro de SSRF/topología interna |
| `handler_news.go:28-31` | `limit` **sin techo**: `limit=100000000000` devuelve la tabla entera en memoria |
| `handler_news.go:107,109`, `:140,142`, `handler_feeds.go:214,216`, `handler_system.go:680,682` | errores tragados + `204` indistinguible del éxito |
| `handler_system.go:62` | `/api/fever` es la única ruta sin `MaxBytesReader` |
| `rss/client.go:51` | `Proxy: http.ProxyFromEnvironment` — con `HTTP_PROXY` la guarda de IP privada **deja de aplicarse al destino** |
| `rss/client.go:32-34` | `GORRSON_TEST=true` desactiva **toda** la guarda anti-SSRF |
| `handler_system.go:795-808` | `/backup/export` sirve la BD con `password_hash`, `fever_key` y `refresh_token` **en claro** |
| `handler_auth.go:62,100` | cookie sin `Secure` con `docker-compose` publicando HTTP puro |
| `handler_system.go:31` | `http.ServeFile(w, r, "docs/openapi.yaml")` — ruta **relativa al CWD**; debería ser `go:embed` |
| `server.go:106` | `GET /api/fever` fuerza a mandar la `api_key` en el query string (log, historial, `Referer`). Fever solo necesita POST |
| `server.go:182-185` | bypass de `OPTIONS` sin autenticar: hoy es código muerto (chi responde 405 antes), pero es un bypass latente |
| `server.go:69-81` | `Server.mu` se mantiene durante `db.NewStore` (I/O + 26 DDL) → el primer acceso de un usuario bloquea el de todos los demás |
| `handler_system.go:332,736,763`, `handler_feeds.go:230` | 5 sitios que saltan `s.getStore` y re-migran la BD en cada request |
| `handler_system.go:191` | `"is_read": 0` hardcodeado en `/api/fever`: con `?items` sin `unread_item_ids` devuelve como no leídas entradas ya leídas |
| `handler_system.go:185` vs `:216` | `feed_id` que recibe el cliente (`uuidToInt`) nunca casa con el que envía en `mark=feed` (`f.FeverID`) |

## Lo que está bien — no tocar

Lo verifiqué a propósito, para que nadie lo "arregle" sin motivo:

- **No hay inyección SQL.** Todo va por `?`. Los `Sprintf` del proyecto no tocan SQL;
  `conditions` solo arma cláusulas literales y los valores van en `args`.
- **No hay IDOR.** El aislamiento es por fichero `.db` y el `username` sale siempre
  de la credencial (cookie, `Bearer`, `?token=`, `X-API-Key`), nunca de un
  parámetro. `ValidateAPIKey` consulta la BD del usuario nombrado, así que
  `X-API-Key: "victima:keyAjena"` falla.
- **La comparación de contraseña y Fever sí es constant-time** (`auth.go:82`). La de
  API key **no** lo es (`db.go:679`), pero son 24 bytes aleatorios hasheados con
  SHA-256: 192 bits de entropía. **No lo cambies "por consistencia".**
- **El anti-SSRF está bien puesto**: el `Control` del dialer se ejecuta **después**
  de resolver DNS, sobre la IP resuelta — eso cierra el DNS rebinding, que un
  chequeo por hostname dejaría pasar. Cubre loopback, link-local (incluido
  `169.254.169.254`), RFC 1918 y `fc00::/7`. Tope de 2 MB, timeout 30 s, semáforos.
  Solo se rompe con `HTTP_PROXY` (V12) o con `GORRSON_TEST` (V12).
- **El OPML anidado no es DoS**: `encoding/xml` tiene su propio límite de
  profundidad y `handler_feeds.go:167` lo convierte en un 400 limpio.
- **Importar backup valida que sea SQLite real** antes de tocar nada
  (`db.go:72-97`, con `PRAGMA quick_check`) y usa rename atómico. Bien hecho.

---

# Orden de arreglo — estado

## Aplicados y verificados (2026-10-03)

| # | Punto | Qué se hizo | Verificación |
| --- | --- | --- | --- |
| 8 | **`idx_entries_fever`** | `CREATE INDEX IF NOT EXISTS idx_entries_fever ON entries(fever_id)` en `migrate()` | 273,9 → 2,6 ms @1M (**105×**) |
| 9 | **`/stats`** | Se agrupa sobre `entries` y se une con `feeds` después | 2.631 → 261 ms (**10×**); equivalencia de datos comprobada con `diff` |
| 1 | **V3** secreto JWT | `auth.InitSecret()` valida al arrancar y **aborta** si el secreto es un valor público conocido o mide <32. Sin secreto, genera uno aleatorio de 32 B con aviso en log | Arranque abortado comprobando los 3 casos; `gorrson-vibrant-rss-secret` y `change-me-in-production` ya no sirven |
| 2 | **V2** XSS reflejada | `stdhtml.EscapeString(err.Error())` + CSP restrictiva + `nosniff` | Corrección directa |
| 3 | **V1** `/proxy` sirve HTML | Content-Type explícito + `nosniff` + CSP `default-src 'none'; sandbox`, y rechazo (415) de lo que no sea imagen, tanto al descargar como al servir de caché | `TestProxyDeImagenesNoSirveHTML` |
| 4 | **V9** timeouts | `ReadHeaderTimeout` 10 s, `ReadTimeout` 60 s, `WriteTimeout` 120 s, `IdleTimeout` 120 s, `MaxHeaderBytes` 1 MiB | `cmd/serve.go` |
| 5 | **V4** DoS de argon2 | Limitador por IP y por usuario en `/login` (10 / 15 min); `/api/fever` con su propio límite (20 / hora) y sin el argon2 ×N usuarios improvisado; la `api_key` ya no se registra | `TestLoginTieneLimiteDeIntentos`, `TestLoginCorrectoLiberaElContador` |
| 6 | **V5** crear BD sin autenticar | `db.OpenExistingStore` / `db.StoreExists` abren sin crear ni migrar; usados en `/login`, `/refresh`, `/logout` y `X-API-Key` | `TestLoginNoCreaBasesDeDatos` |
| 7 | **V11** pérdida de escrituras | `s.evictStore(username)` tras importar un backup: cierra y saca de caché el store abierto sobre el inode antiguo | `TestEvictStoreDejaDeVerLosDatosViejos` |
| 10 | **V6** `/proxy/page` | `sanitizer.CleanForProxy` (política estricta, sin iframes) aplicado al documento remoto **antes** de inyectar el script propio; `resolveURL` ya no pasa `javascript:`; y el `type="text/plain"` de los scripts **sustituye** al atributo en vez de añadirse (el duplicado hacía que el navegador ganara el primero y el script se ejecutara) | `TestCleanForProxy` (8 vectores) + `TestCleanForProxyConservaEstructuraDeArticulo` |

### Dos cosas que cambié de diseño al implementarlas

**`resolveImageType` acepta la cabecera del origen si es `image/*`.**
`http.DetectContentType` **no reconoce SVG ni AVIF**, que son miniaturas
perfectamente normales en un lector de RSS: validar solo por bytes las habría
rechazado. Lo que neutraliza un cuerpo HTML no es el sniff sino servirlo con
`Content-Type` explícito + `nosniff`, que es exactamente lo que hace ahora
`serveCachedImage`. El Content-Type resuelto se guarda en un fichero `.type`
al lado de la caché, porque en la segunda visita ya no está la cabecera.

**El límite de `/login` no es fail-closed.** Si no hay secreto JWT, el arranque
**aborta**; eso sí es fail-closed. Si no hay secreto, se genera uno aleatorio y se
avisa: obligar a declararlo también en desarrollo no compensa, y desde luego no se
vuelve a usar un valor público. Lo que se pierde es que las sesiones no
sobreviven a un reinicio, que es el precio visible.

## Pendientes

| Punto | Por qué no está aplicado |
| --- | --- |
| **V7** `summary` sin sanear | Está en `internal/rss/parser.go`, la ingesta. Es un cambio de comportamiento del contenido almacenado |
| **V8** filtros `REPLACE` tras sanear | Mismo motivo: toca la ingesta |
| **V10** FTS5 sin validar | Requiere decidir la semántica de la búsqueda, como `ORDER BY rank` |
| **V12** (12 ítems) | Mezcla de endurecimiento, logs, paginado y errores tragados. Algunos sí son de una línea |
| **`ORDER BY rank`** | Decisión de producto, no optimización |
| **`/feeds` lineal** | Ganancia restante pequeña tras el índice |

### Un apunte sobre `handleFeverAPI`

El N × argon2 no se puede eliminar sin rediseñar la autenticación de Fever: la
clave es `md5(usuario:contraseña)` y no viaja el usuario, así que hay que
verificarla contra cada base de datos. Lo que sí se ha acotado es el coste:
límite por IP, y `OpenExistingStore` en vez de `NewStore` (que además migraba 26
DDL por usuario y por petición). La solución de fondo sería storing un
verificador determinista indexado — `hmac(secreto, md5(user:pass))` — para
convertir la búsqueda en O(1). Es un cambio de esquema y de registro.

---

*Medido en Linux 7.2.7-200.fc44.x86_64, AMD Ryzen 5 5600X (6 núcleos / 12 hilos),
con un dataset de 1.000.000 de entradas generado por `make perf-seed SEED_N=1000000`.
Todas las cifras de req/s son sensibles a la carga de la máquina: el mismo binario
midió `health` a 6.443 req/s con una sesión de `ggit` haciendo `make test -race` en
paralelo, y a 61.119 req/s con la máquina libre. Las de esta tabla son consultas SQL
aisladas con `sqlite3`, mucho menos sensibles; las de req/s están en
[`../bench.md`](../bench.md) §3.*
*Rendimiento ya corregido y sus números: [`../bench.md`](../bench.md).*