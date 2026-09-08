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
