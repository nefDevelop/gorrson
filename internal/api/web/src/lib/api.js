/** @type {Promise<string> | null} */
let refreshPromise = null;

/**
 * @param {string} url
 * @param {RequestInit & { headers?: Record<string, string> }} [options]
 */
export async function fetchWithAuth(url, options = {}) {
  let token = localStorage.getItem("token");
  /** @type {Record<string, string>} */
  let headers = { ...options.headers };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  let res = await fetch(url, { ...options, headers });

  if (res.status === 401) {
    const refreshToken = localStorage.getItem("refresh_token");
    const username = localStorage.getItem("username");

    if (refreshToken && username) {
      if (!refreshPromise) {
        refreshPromise = fetch("/refresh", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken, username }),
        })
          .then(async (r) => {
            if (!r.ok) {
              localStorage.removeItem("token");
              localStorage.removeItem("refresh_token");
              throw new Error("Refresh failed");
            }
            const data = await r.json();
            localStorage.setItem("token", data.access_token);
            return data.access_token;
          })
          .finally(() => {
            refreshPromise = null;
          });
      }

      try {
        token = await refreshPromise;
        headers["Authorization"] = `Bearer ${token}`;
        res = await fetch(url, { ...options, headers });
      } catch (e) {
        // Si falla el refresh, dejamos que pase el 401 original
      }
    }
  }

  if (res.status === 401) throw new Error("Unauthorized");

  return res;
}

/**
 * Helper to clean and build query strings
 * @param {Record<string, any>} params
 */
function buildQuery(params) {
  const clean = Object.fromEntries(Object.entries(params).filter(([, v]) => v != null && v !== "" && v !== "null"));
  return new URLSearchParams(clean).toString();
}

/**
 * @param {Record<string, any>} [params]
 */
export async function fetchNews(params = {}) {
  const res = await fetchWithAuth(`/news?${buildQuery(params)}`);
  if (!res.ok) throw new Error("Failed to fetch news");
  return res.json();
}

export async function fetchFeeds() {
  const res = await fetchWithAuth("/feeds");
  if (!res.ok) throw new Error("Failed to fetch feeds");
  return res.json();
}

/**
 * @param {string | number} id
 */
export async function markAsRead(id) {
  const res = await fetchWithAuth(`/news/${id}/read`, { method: "PUT" });
  return res.ok;
}

/**
 * @param {Record<string, any>} [params]
 */
export async function markAllRead(params = {}) {
  const res = await fetchWithAuth(`/news/read-all?${buildQuery(params)}`, { method: "PUT" });
  return res.ok;
}

/**
 * @param {string | number} id
 * @param {boolean} [favorite]
 */
export async function toggleFavorite(id, favorite = true) {
  const res = await fetchWithAuth(`/news/${id}/favorite?favorite=${favorite}`, { method: "PUT" });
  return res.ok;
}

export async function fetchFilters() {
  const res = await fetchWithAuth("/filters");
  if (!res.ok) throw new Error("Failed to fetch filters");
  return res.json();
}

/**
 * @param {any} filter
 */
export async function addFilter(filter) {
  const res = await fetchWithAuth("/filters", {
    method: "POST",
    body: JSON.stringify(filter),
    headers: { "Content-Type": "application/json" },
  });
  return res.ok;
}

/**
 * @param {string | number} id
 */
export async function deleteFilter(id) {
  const res = await fetchWithAuth(`/filters/${id}`, { method: "DELETE" });
  return res.ok;
}

export async function exportConfig() {
  const res = await fetchWithAuth("/config/export");
  if (!res.ok) throw new Error("Failed to export config");
  
  // Convertir la respuesta a Blob y forzar su descarga en el navegador
  const blob = await res.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "gorrson-config.json";
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(url);
}

/**
 * @param {Object} configData
 */
export async function importConfig(configData) {
  const res = await fetchWithAuth("/config/import", {
    method: "POST",
    body: JSON.stringify(configData),
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to import config");
  return res.json();
}

/**
 * @param {string} username
 * @param {string} password
 */
export async function login(username, password) {
  const res = await fetch("/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("Invalid credentials");
  const data = await res.json();
  localStorage.setItem("token", data.access_token);
  localStorage.setItem("refresh_token", data.refresh_token);
  localStorage.setItem("username", username);
  return data;
}

/**
 * @param {string} url
 * @param {string} [category]
 */
export async function addFeed(url, category = "") {
  const res = await fetchWithAuth("/feeds", {
    method: "POST",
    body: JSON.stringify({ url, category }),
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "Failed to add feed");
  }
  try {
    return await res.json();
  } catch {
    return {};
  }
}

/**
 * @param {string | number} id
 * @param {any} data
 */
export async function updateFeed(id, data) {
  const res = await fetchWithAuth(`/feeds/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to update feed");
  return res.json();
}

/**
 * @param {string | number} id
 */
export async function deleteFeed(id) {
  const res = await fetchWithAuth(`/feeds/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Failed to delete feed");
}

// Cola de concurrencia para proteger el límite (Semaphore) del servidor
const MAX_CONCURRENT_FETCHES = 3;
let activeFetches = 0;
/** @type {Array<(value?: any) => void>} */
let fetchQueue = [];

/**
 * @template T
 * @param {() => Promise<T>} fn
 * @returns {Promise<T>}
 */
async function enqueueFetch(fn) {
  if (activeFetches >= MAX_CONCURRENT_FETCHES) {
    await new Promise((resolve) => fetchQueue.push(resolve));
  }
  activeFetches++;
  try {
    return await fn();
  } finally {
    activeFetches--;
    if (fetchQueue.length > 0) {
      const next = fetchQueue.shift();
      if (next) next();
    }
  }
}

/**
 * @param {string | number} id
 */
export async function fetchFullContent(id) {
  return enqueueFetch(async () => {
    const res = await fetchWithAuth(`/news/${id}/content`);
    if (!res.ok) throw new Error("Failed to fetch full content");
    return res.text();
  });
}

export async function fetchAPIKeys() {
  const res = await fetchWithAuth("/api-keys");
  if (!res.ok) throw new Error("Failed to fetch API keys");
  return res.json();
}

/**
 * @param {string} name
 */
export async function createAPIKey(name) {
  const res = await fetchWithAuth("/api-keys", {
    method: "POST",
    body: JSON.stringify({ name }),
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to create API key");
  return res.json();
}

/**
 * @param {string | number} id
 */
export async function deleteAPIKey(id) {
  const res = await fetchWithAuth(`/api-keys/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Failed to delete API key");
}

export async function fetchStats() {
  const res = await fetchWithAuth("/stats");
  if (!res.ok) throw new Error("Failed to fetch stats");
  return res.json();
}

export async function updateRecipes() {
  const res = await fetchWithAuth("/recipes/update", { method: "POST" });
  return res.ok;
}
