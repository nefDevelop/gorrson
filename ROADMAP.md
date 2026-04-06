# Gorrson Roadmap

Este documento detalla la visión a corto y largo plazo para el lector RSS `gorrson`, enfocándose en la simplicidad, privacidad y el soporte de fuentes modernas sin necesidad de scrapers complejos.

## 🚀 Fase 1: Integración de Redes Sociales (Vía RSS Nativo/Bridges)
El objetivo es permitir que `gorrson` actúe como un cliente unificado de lectura para redes sociales aprovechando protocolos abiertos.

### 🐘 Mastodon & Fediverso (Nativo)
- [ ] **Soporte de perfiles:** Documentar y asegurar la compatibilidad con `https://instancia/@usuario.rss`.
- [ ] **Soporte de Hashtags:** Añadir feeds de etiquetas (ej. `https://instancia/tags/golang.rss`).
- [ ] **Mejora del Parser:** Extraer avatares de usuario y adjuntos (imágenes/video) que Mastodon incluye en el XML.

### 🔖 Reddit (Nativo)
- [ ] **Soporte de Subreddits:** Documentar el uso de `.rss` al final de cualquier URL de subreddit.
- [ ] **Soporte de Usuarios:** Seguimiento de perfiles individuales de Reddit.
- [ ] **Soporte de Búsquedas:** Integrar feeds de búsqueda filtrada por relevancia o fecha.

### 🐦 X (Twitter) & Bluesky (Vía Bridges)
- [ ] **Integración con RSS-Bridge:** Crear una guía o preset dentro de `gorrson` para añadir fácilmente URLs de instancias públicas de RSS-Bridge para X.
- [ ] **Soporte Bluesky:** Utilizar el endpoint nativo/comunitario de RSS para perfiles de Bluesky.

---

## 🛠️ Fase 2: Mejoras en el Motor de RSS
Para que las redes sociales se vean bien, el motor actual necesita ajustes finos:

- [ ] **Gestión de User-Agents Personalizables:** Permitir cambiar el `UserAgent` por feed para evitar bloqueos de plataformas como Reddit o X (vía bridge).
- [ ] **Extracción de Imágenes Mejorada:** Refinar `internal/rss/parser.go` para detectar imágenes dentro de etiquetas `media:content` o `enclosure` comunes en redes sociales.
- [ ] **Soporte de Autenticación Básica:** Para feeds privados o protegidos por contraseña.

---

## 📱 Fase 3: Interfaz de Usuario (Web UI)
- [ ] **Vista Estilo "Feed":** Una opción de visualización más compacta para posts de redes sociales (similar a un timeline).
- [ ] **Filtros por Red Social:** Poder ver solo posts de "Reddit", "Mastodon", etc., mediante etiquetas automáticas.

---

## 🧪 Próximos Pasos Inmediatos
1. Probar la sincronización actual con un feed de Reddit en `internal/rss/parser_test.go`.
2. Verificar si el `sanitizer.go` actual limpia correctamente el HTML que envían las redes sociales (a menudo vienen con muchos tags de estilo).
