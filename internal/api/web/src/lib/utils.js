/**
 * @param {string | number | Date} dateString
 */
export function formatTimeAgo(dateString) {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 5) {
    return `just now`;
  }

  if (seconds < 60) {
    return `${seconds}s ago`;
  }

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }

  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours}h ago`;
  }

  const days = Math.floor(hours / 24);
  if (days < 7) {
    return `${days}d ago`;
  }

  return date.toLocaleDateString(); // Fallback for older items
}

/**
 * @param {string} html
 */
export function stripHtml(html) {
  if (!html) return "";
  return html
    .replace(/<[^>]*>?/gm, "")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .trim();
}

/**
 * @param {string | null | undefined} url
 * @returns {string}
 */
export function proxyImage(url) {
  if (!url) return '';
  return `/proxy?url=${encodeURIComponent(url)}`;
}

/**
 * Reescribe un atributo srcset para que cada candidato pase por /proxy.
 *
 * El saneador del servidor solo valida el esquema de src y href, así que el
 * srcset llega intacto: sin este paso las variantes de resolución cargan
 * directamente del origen (se saltan la caché y la protección SSRF) y una URL
 * con un esquema no permitido (p. ej. file:) llegaría viva al navegador.
 *
 * @param {string | null | undefined} value
 * @returns {string}
 */
export function proxySrcset(value) {
  return (value || "")
    .split(",")
    .map((candidate) => {
      const trimmed = candidate.trim();
      const parts = trimmed.split(/\s+/);
      const url = parts[0];
      if (!trimmed || url.startsWith("data:") || url.startsWith("/proxy?")) {
        return trimmed;
      }
      parts[0] = proxyImage(url);
      return parts.join(" ");
    })
    .filter(Boolean)
    .join(", ");
}

/**
 * @param {string | null | undefined} html
 * @returns {string}
 */
export function readingTime(html) {
  const text = stripHtml(html || '');
  if (!text) return '';
  const words = text.split(/\s+/).length;
  const mins = Math.ceil(words / 200);
  return mins < 1 ? '<1 min' : `${mins} min`;
}

/**
 * @param {string} html
 */
export function extractFirstImage(html) {
  if (!html) return null;

  // Fast path: si el texto ni siquiera contiene una etiqueta img, evitamos el pesado DOMParser
  if (!html.toLowerCase().includes("<img")) return null;

  // Use browser's native DOMParser to safely inspect attributes
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  const imgs = doc.querySelectorAll("img");

  for (const img of imgs) {
    const src = img.getAttribute("src");
    if (!src) continue;

    // Ignore images that look like tracking pixels or spacers
    const lowerSrc = src.toLowerCase();
    if (lowerSrc.includes("pixel") || lowerSrc.includes("spacer") || lowerSrc.includes("1x1") || lowerSrc.includes("tracking")) {
      continue;
    }

    // Check dimensions if specified as attributes
    const w = parseInt(img.getAttribute("width") || "101"); // Assume large if not specified
    const h = parseInt(img.getAttribute("height") || "101");

    // Skip small icons or author thumbnails (like the 40x40 ESPN example)
    if (w > 0 && w < 100) continue;
    if (h > 0 && h < 100) continue;

    return src;
  }
  return null;
}
