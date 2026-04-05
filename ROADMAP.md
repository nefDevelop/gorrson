# 🗺️ Gorrson Roadmap: Análisis de Brechas vs. Miniflux

Este documento analiza las funcionalidades que Miniflux posee y Gorrson no, evaluando el "coste" técnico de su implementación para un entorno ligero (Go + Termux).

---

## 1. Descubrimiento Automático (Feed Discovery)
*   **Descripción**: Permitir que el usuario ingrese `https://blog.ejemplo.com` y el motor encuentre el enlace al RSS en el HTML.
*   **Esfuerzo**: Bajo (1-2 días).
*   **Complejidad**: Baja. Requiere un parser de HTML como `goquery`.
*   **Impacto en Recursos**: Mínimo.
*   **Análisis**: Muy recomendado. Evita que el usuario tenga que buscar manualmente el XML en el código fuente.

## 2. Filtrado y Reescritura (Rules Engine)
*   **Descripción**: Reglas basadas en Regex para descartar artículos (ej: "No mostrar si contiene 'Publicidad'") o modificar contenido.
*   **Esfuerzo**: Medio (2-3 días).
*   **Complejidad**: Baja. Se basa en lógica de strings antes de la inserción en DB.
*   **Impacto en Recursos**: Bajo.
*   **Análisis**: Útil para limpiar feeds ruidosos. Fácil de implementar en el worker de Go.

## 3. Extracción de Contenido Completo (Scraping)
*   **Descripción**: Descargar la web original del artículo y extraer solo el texto (estilo "Modo Lectura") cuando el RSS solo envía un resumen.
*   **Esfuerzo**: Medio-Alto (1 semana).
*   **Complejidad**: Alta. Requiere heurísticas para identificar el contenido principal y limpiar el "ruido" (anuncios, menús).
*   **Impacto en Recursos**: **Alto**. Multiplica las peticiones de red y el uso de CPU durante la sincronización.
*   **Análisis**: Transforma la experiencia de lectura, pero debe ser opcional por feed para ahorrar batería en dispositivos móviles.

## 4. Proxy de Imágenes (Privacy Proxy)
*   **Descripción**: Servir las imágenes de los artículos a través de Gorrson para ocultar la IP del usuario al servidor original.
*   **Esfuerzo**: Medio (3-4 días).
*   **Complejidad**: Media. Requiere un endpoint que descargue y re-sirva imágenes, posiblemente con caché.
*   **Impacto en Recursos**: Medio (Uso de disco para caché).
*   **Análisis**: Importante para la privacidad, pero complejo de gestionar si no hay mucho almacenamiento disponible.

## 5. Compatibilidad con APIs de Terceros (Fever/Google Reader)
*   **Descripción**: Implementar los endpoints y el formato JSON que esperan apps como Reeder, FeedMe o NewsBlur.
*   **Esfuerzo**: Alto (2 semanas).
*   **Complejidad**: Alta. Estas APIs son antiguas, usan formatos específicos (XML/JSON) y métodos de autenticación propios.
*   **Impacto en Recursos**: Bajo.
*   **Análisis**: **Estratégico**. Permitiría usar Gorrson con las mejores apps de lectura del mercado sin desarrollar una interfaz propia.

## 6. Soporte WebSub (Real-time)
*   **Descripción**: Recibir notificaciones push cuando un feed se actualiza.
*   **Esfuerzo**: Alto (Requiere infraestructura externa).
*   **Complejidad**: Media. Necesita que el servidor sea accesible desde internet (IP pública o túnel).
*   **Impacto en Recursos**: Bajo (Ahorra batería al no hacer polling constante).
*   **Análisis**: Difícil de implementar si Gorrson corre localmente en un móvil sin una URL pública estable.

---

## 📈 Recomendación de Prioridades

1.  **Feed Discovery**: Ganancia inmediata con poco esfuerzo.
2.  **Filtrado Simple**: Mejora la calidad del contenido.
3.  **API Fever**: Abre la puerta al ecosistema de apps móviles existentes.
