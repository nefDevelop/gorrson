# Gorrson Roadmap


## 🚀 Fase 1: Integración de Redes Sociales (Vía RSS Nativo/Bridges)


### 🐘 Mastodon & Fediverso (Nativo)
- [ ] **Soporte de Hashtags:** Añadir feeds de etiquetas (ej. `https://instancia/tags/golang.rss`).



### 🐦 X (Twitter) & Bluesky (Vía Bridges)
- [ ] **Integración con RSS-Bridge:** Crear una guía o preset dentro de `gorrson` para añadir fácilmente URLs de instancias públicas de RSS-Bridge para X.
- [ ] **Soporte Bluesky:** Utilizar el endpoint nativo/comunitario de RSS para perfiles de Bluesky.

---

## 🛠️ Fase 2: Mejoras en el Motor de RSS
Para que las redes sociales se vean bien, el motor actual necesita ajustes finos:

- [ ] **Soporte de Autenticación Básica:** Para feeds privados o protegidos por contraseña.

---

## 🌐 Fase 4: Ecosistema y Filtrado Avanzado
El objetivo es abrir Gorrson al ecosistema de apps móviles y permitir un control total sobre el orden del feed según la importancia.

### 📱 Ecosistema Móvil & APIs
- [ ] **Soporte Fever API:** Implementar un endpoint compatible para permitir la sincronización con apps como Reeder, NetNewsWire o Fiery Feeds.
- [ ] **Soporte Google Reader API (GReader):** Alternativa para mayor compatibilidad con clientes legacy.

### ⚡ Filtrado por Prioridad (Control del Usuario)
- [ ] **Acción "FIRST" o "IMPORTANT":** Añadir una regla de filtrado que mueva automáticamente los posts marcados a la parte superior del feed, ignorando la cronología estándar.
- [ ] **Acción "NOTIFY":** Envío de notificaciones push para términos específicos.

### 🧩 Extensibilidad (Motor Dinámico)
- [ ] **Motor de Scrapers por Recetas:** Permitir la extracción de contenido mediante selectores CSS definidos en archivos YAML externos.

