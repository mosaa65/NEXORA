const API_BASE = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");

// Catalogue data changes far less often than users navigate between pages. Keep
// successful GET responses briefly in memory and session storage so back/forward
// navigation (and an accidental page refresh) does not repeatedly hit the API.
const READ_CACHE_TTL = 2 * 60 * 1000;
const HEALTH_CACHE_TTL = 15 * 1000;
const CACHE_PREFIX = "nexora:api-cache:";
const responseCache = new Map();
const pendingRequests = new Map();

function cacheTTL(path) {
  return path.startsWith("/api/health") ? HEALTH_CACHE_TTL : READ_CACHE_TTL;
}

function isCacheableRequest(path, options) {
  return (!options.method || options.method.toUpperCase() === "GET")
    && !path.startsWith("/api/admin/")
    && !path.startsWith("/api/stream/");
}

function readCachedResponse(key) {
  const now = Date.now();
  const memory = responseCache.get(key);
  if (memory && memory.expiresAt > now) return memory.data;
  if (memory) responseCache.delete(key);

  try {
    const saved = JSON.parse(sessionStorage.getItem(`${CACHE_PREFIX}${key}`));
    if (saved?.expiresAt > now) {
      responseCache.set(key, saved);
      return saved.data;
    }
    sessionStorage.removeItem(`${CACHE_PREFIX}${key}`);
  } catch {}
  return undefined;
}

function saveCachedResponse(key, data, ttl) {
  const entry = { data, expiresAt: Date.now() + ttl };
  responseCache.set(key, entry);
  try { sessionStorage.setItem(`${CACHE_PREFIX}${key}`, JSON.stringify(entry)); } catch {}
}

// Exported for admin save/delete workflows and future live-refresh events.
export function invalidateAPICache() {
  responseCache.clear();
  try {
    Object.keys(sessionStorage)
      .filter((key) => key.startsWith(CACHE_PREFIX))
      .forEach((key) => sessionStorage.removeItem(key));
  } catch {}
}

async function requestJSON(path, options = {}) {
  const cacheable = isCacheableRequest(path, options);
  const cacheKey = `${API_BASE}${path}`;
  if (cacheable) {
    const cached = readCachedResponse(cacheKey);
    if (cached !== undefined) return cached;
    const pending = pendingRequests.get(cacheKey);
    if (pending) return pending;
  }

  const performRequest = async () => {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {})
    },
    ...options
  });

  const text = await response.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: text };
    }
  }

  if (!response.ok) {
    const error = new Error(data?.error || response.statusText || `Request failed with status ${response.status}`);
    error.status = response.status;
    error.payload = data;
    throw error;
  }

  if (cacheable) {
    saveCachedResponse(cacheKey, data, cacheTTL(path));
  } else if ((options.method || "GET").toUpperCase() !== "GET") {
    // Any successful write can affect catalogue summaries, counts and details.
    invalidateAPICache();
  }
  return data;
  };

  if (!cacheable) return performRequest();
  const pending = performRequest().finally(() => pendingRequests.delete(cacheKey));
  pendingRequests.set(cacheKey, pending);
  return pending;
}

export async function getHealth() {
  return requestJSON("/api/health");
}

export async function getCategories() {
  return requestJSON("/api/categories");
}

export async function createCategory(data) {
  return requestJSON("/api/categories", {
    method: "POST",
    body: JSON.stringify(data)
  });
}

export async function updateCategory(id, data) {
  return requestJSON(`/api/categories/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(data)
  });
}

export async function deleteCategory(id) {
  return requestJSON(`/api/categories/${encodeURIComponent(id)}`, {
    method: "DELETE"
  });
}

export async function searchLibrary(query, options = {}) {
  const params = new URLSearchParams();
  if (query !== undefined) {
    params.set("q", query);
  }
  if (options.limit) {
    params.set("limit", String(options.limit));
  }
  if (options.type) {
    params.set("type", options.type);
  }
  if (options.category) {
    params.set("category", options.category);
  }
  return requestJSON(`/api/search?${params.toString()}`);
}

export async function syncIndex(limit = 1000) {
  return requestJSON(`/api/search/sync?limit=${limit}`, {
    method: "POST"
  });
}

export async function indexLibrary(roots) {
  return requestJSON("/api/index", {
    method: "POST",
    body: JSON.stringify({ roots })
  });
}

export async function previewIndex(roots) {
  return requestJSON("/api/index/preview", {
    method: "POST",
    body: JSON.stringify({ roots })
  });
}

export async function classifyOriginsFromFolders() {
  return requestJSON("/api/library/classify-origins", { method: "POST" });
}

export async function getDuplicateGroups() { return requestJSON("/api/library/duplicates"); }
export async function getMissingEpisodes() { return requestJSON("/api/library/missing-episodes"); }
export async function getQualityReport() { return requestJSON("/api/quality/report"); }
export async function calculateChecksums(mediaItemId) { return requestJSON("/api/media/checksums", { method: "POST", body: JSON.stringify(mediaItemId ? { mediaItemId } : {}) }); }

export async function previewMigration(root) {
  return requestJSON("/api/migration/preview", {
    method: "POST",
    body: JSON.stringify({ root })
  });
}

export async function copyMedia(request) {
  return requestJSON("/api/migration/copy", {
    method: "POST",
    body: JSON.stringify(request)
  });
}

export async function getMediaFiles(mediaId) {
  if (!mediaId) {
    return { count: 0, files: [] };
  }
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/files`);
}

export async function getMediaList(options = {}) {
  const params = new URLSearchParams();
  if (options.category) params.set("category", options.category);
  if (options.type) params.set("type", options.type);
  if (options.q) params.set("q", options.q);
  if (options.sort) params.set("sort", options.sort);
  if (options.limit) params.set("limit", String(options.limit));
  if (options.offset) params.set("offset", String(options.offset));
  return requestJSON(`/api/media?${params.toString()}`);
}

// Provider franchises are read from NEXORA's local database. This request
// never asks TMDB for data while the user is browsing.
export async function getFranchises(limit = 12) {
  return requestJSON(`/api/franchises?limit=${encodeURIComponent(limit)}`);
}

export async function getFranchise(slug) {
  return requestJSON(`/api/franchises/${encodeURIComponent(slug)}`);
}
export async function getFranchiseMedia(slug, options = {}) {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", String(options.limit));
  if (options.sort) params.set("sort", options.sort);
  return requestJSON(`/api/franchises/${encodeURIComponent(slug)}/media?${params.toString()}`);
}
export async function refreshFranchise(id) {
  return requestJSON(`/api/admin/franchises/${encodeURIComponent(id)}/refresh`, { method: "POST" });
}
export async function refreshMissingFranchises(limit = 24) {
  return requestJSON(`/api/admin/franchises/refresh-missing?limit=${encodeURIComponent(limit)}`, { method: "POST" });
}

// The showcase endpoint is database-first: editorial collections and media
// summaries have already been persisted by NEXORA before the UI renders them.
export async function getShowcases(options = {}) {
  const params = new URLSearchParams();
  if (options.context) params.set("context", options.context);
  if (options.category) params.set("category", options.category);
  if (options.limit) params.set("limit", String(options.limit));
  return requestJSON(`/api/showcases?${params.toString()}`);
}

export async function getSmartHubs(scope) {
  const params = new URLSearchParams();
  if (scope) params.set("scope", scope);
  return requestJSON(`/api/hubs?${params.toString()}`);
}
export async function getSmartHub(slug) { return requestJSON(`/api/hubs/${encodeURIComponent(slug)}`); }
export async function getSmartHubMedia(slug, options = {}) {
  const params = new URLSearchParams();
  if (options.sort) params.set("sort", options.sort);
  if (options.limit) params.set("limit", String(options.limit));
  if (options.offset) params.set("offset", String(options.offset));
  return requestJSON(`/api/hubs/${encodeURIComponent(slug)}/media?${params.toString()}`);
}
export async function getAdminSmartHubs() { return requestJSON("/api/admin/hubs"); }
export async function saveSmartHub(slug, data) { return requestJSON(`/api/admin/hubs/${encodeURIComponent(slug)}`, { method: "PUT", body: JSON.stringify(data) }); }
export async function createSmartHub(data) { return requestJSON("/api/admin/hubs", { method: "POST", body: JSON.stringify(data) }); }

export async function getCollections() { return requestJSON("/api/admin/collections"); }
export async function saveCollection(data) {
  const method = data.id ? "PUT" : "POST";
  const path = data.id ? `/api/admin/collections/${data.id}` : "/api/admin/collections";
  return requestJSON(path, { method, body: JSON.stringify(data) });
}
export async function deleteCollection(id) { return requestJSON(`/api/admin/collections/${id}`, { method: "DELETE" }); }

export async function getMediaDetail(mediaId) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}`);
}

export async function getMediaMetadataSnapshot(mediaId, locale = "ar-SA") {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/metadata/raw?locale=${encodeURIComponent(locale)}`);
}

export async function getMediaSeasonMetadata(mediaId, locale = "ar-SA") {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/metadata/seasons?locale=${encodeURIComponent(locale)}`);
}

export async function enrichMedia(mediaId, options = {}) {
  const selected = options.tmdbId;
  const query = selected ? `?tmdb_id=${encodeURIComponent(selected)}` : "";
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/enrich${query}`, {
    method: "POST"
  });
}

export async function searchTMDBCandidates({ title, type, year }) {
  const params = new URLSearchParams({ title, type: type || "movie" });
  if (year) params.set("year", String(year));
  return requestJSON(`/api/tmdb/candidates?${params.toString()}`);
}

export async function createMediaItem(data) {
  return requestJSON("/api/media", {
    method: "POST",
    body: JSON.stringify(data)
  });
}

export async function updateMediaItem(mediaId, data) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}`, {
    method: "PUT",
    body: JSON.stringify(data)
  });
}

export async function deleteMediaItem(mediaId) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}`, {
    method: "DELETE"
  });
}

export async function updateMediaMetadata(mediaId, metadata) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/metadata`, {
    method: "PUT",
    body: JSON.stringify(metadata)
  });
}

export async function getDashboardStats() {
  return requestJSON("/api/dashboard/stats");
}

export async function getDisks() {
  return requestJSON("/api/disks");
}

export async function scanDisks() {
  return requestJSON("/api/disks/scan", { method: "POST" });
}

// TMDB Integration API
export async function getTMDBSettings() {
  return requestJSON("/api/tmdb/settings");
}

export async function updateTMDBSettings(settings) {
  return requestJSON("/api/tmdb/settings", {
    method: "PUT",
    body: JSON.stringify(settings)
  });
}

export async function getTMDBStats() {
  return requestJSON("/api/tmdb/stats");
}
export async function getTMDBUsageHistory(days = 90) { return requestJSON(`/api/tmdb/usage/history?days=${encodeURIComponent(days)}`); }

export async function getTMDBModules() {
  return requestJSON("/api/tmdb/modules");
}

export async function getTMDBQueue() { return requestJSON("/api/tmdb/queue"); }
export async function enqueueTMDBRefresh(mediaItemId, priority = 0) {
  return requestJSON("/api/tmdb/queue", { method: "POST", body: JSON.stringify({ media_item_id: mediaItemId, priority }) });
}
export async function cancelTMDBQueueJob(id) { return requestJSON(`/api/tmdb/queue/${encodeURIComponent(id)}/cancel`, { method: "POST" }); }

export async function updateTMDBModules(payload) {
  return requestJSON("/api/tmdb/modules", {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function testTMDBConnection() {
  return requestJSON("/api/tmdb/test", {
    method: "POST"
  });
}

export async function getTMDBRemoteConfiguration() {
  return requestJSON("/api/tmdb/configuration");
}

export async function getTMDBPreview(mediaId) {
  return requestJSON(`/api/tmdb/preview/${encodeURIComponent(mediaId)}`);
}

// System Directory Explorer API (Browse Windows Drives D:\, E:\)
export async function getSystemDrives() {
  return requestJSON("/api/system/drives");
}

export async function browseSystemDirectory(path) {
  const params = path ? `?path=${encodeURIComponent(path)}` : "";
  return requestJSON(`/api/system/browse${params}`);
}

// Admin Authentication API
export async function adminLogin(username, password) {
  return requestJSON("/api/admin/login", {
    method: "POST",
    body: JSON.stringify({ username, password })
  });
}

export async function checkAdminSession() {
  const token = localStorage.getItem("nexora_admin_token") || "";
  return requestJSON("/api/admin/session", {
    headers: { Authorization: `Bearer ${token}` }
  });
}

export async function adminLogout() {
  localStorage.removeItem("nexora_admin_token");
  localStorage.removeItem("nexora_admin_user");
  return requestJSON("/api/admin/logout", { method: "POST" });
}



export async function getFileSubtitles(fileId) {
  return requestJSON(`/api/stream/file/${encodeURIComponent(fileId)}/subtitles`);
}

// All catalogue graph reads are local API reads. TMDB is used only by the
// explicit enrichment workflow on the server.
export async function getPeople(limit = 18) {
  return requestJSON(`/api/people?limit=${encodeURIComponent(limit)}`);
}

export async function getPerson(slug) {
  return requestJSON(`/api/people/${encodeURIComponent(slug)}`);
}

export async function getPersonMedia(slug, options = {}) {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", String(options.limit));
  if (options.sort) params.set("sort", options.sort);
  const query = params.toString();
  return requestJSON(`/api/people/${encodeURIComponent(slug)}/media${query ? `?${query}` : ""}`);
}

export function resolveAPIURL(path) {
  if (!path) {
    return "";
  }
  if (/^(?:https?:\/\/|data:|blob:)/i.test(path)) {
    return path;
  }
  return `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
}
