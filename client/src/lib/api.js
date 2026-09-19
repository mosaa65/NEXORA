const API_BASE = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");

// Fast in-memory cache with smart TTLs and zero JSON serialization overhead.
// Instant (0ms) access when navigating back and forth across catalogue surfaces.
const READ_CACHE_TTL = 3 * 60 * 1000;       // 3 minutes for catalogue listings
const DETAIL_CACHE_TTL = 8 * 60 * 1000;     // 8 minutes for media details & snapshots
const STATIC_CACHE_TTL = 15 * 60 * 1000;    // 15 minutes for categories, hubs, showcases
const HEALTH_CACHE_TTL = 15 * 1000;         // 15 seconds for health
const MAX_CACHE_ENTRIES = 250;

const responseCache = new Map();
const pendingRequests = new Map();

function cacheTTL(path) {
  if (path.startsWith("/api/health")) return HEALTH_CACHE_TTL;
  if (
    path.startsWith("/api/categories") ||
    path.startsWith("/api/showcases") ||
    path.startsWith("/api/hubs") ||
    path.startsWith("/api/franchises") ||
    path.startsWith("/api/people")
  ) {
    return STATIC_CACHE_TTL;
  }
  if (path.startsWith("/api/media/") || path.startsWith("/api/stream/file/")) {
    return DETAIL_CACHE_TTL;
  }
  return READ_CACHE_TTL;
}

function isCacheableRequest(path, options = {}) {
  return (!options.method || options.method.toUpperCase() === "GET")
    && options.cache !== "no-store"
    && !path.startsWith("/api/admin/")
    && !path.startsWith("/api/stream/")
    && !path.startsWith("/api/transfer/");
}

function readCachedResponse(key) {
  const memory = responseCache.get(key);
  if (memory) {
    if (memory.expiresAt > Date.now()) {
      return memory.data;
    }
    responseCache.delete(key);
  }
  return undefined;
}

function saveCachedResponse(key, data, ttl) {
  if (responseCache.size >= MAX_CACHE_ENTRIES && !responseCache.has(key)) {
    const oldestKey = responseCache.keys().next().value;
    if (oldestKey) responseCache.delete(oldestKey);
  }
  responseCache.set(key, { data, expiresAt: Date.now() + ttl });
}

// Exported for admin save/delete workflows and future live-refresh events.
export function invalidateAPICache() {
  responseCache.clear();
  pendingRequests.clear();
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
    let authHeader = {};
    if (typeof localStorage !== "undefined") {
      const token = localStorage.getItem("nexora_admin_token");
      if (token) {
        authHeader = { Authorization: `Bearer ${token}` };
      }
    }
    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...authHeader,
        ...(options.headers || {})
      }
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

export async function indexLibrary(roots, options = {}) {
  const body = { roots };
  // The scan endpoint understands mode/inspect/syncSearch. Sending them
  // explicitly is what lets the UI offer "full" versus "incremental" rather
  // than always running the server's default.
  if (options.mode) body.mode = options.mode;
  if (options.inspect !== undefined) body.inspect = options.inspect;
  if (options.syncSearch !== undefined) body.syncSearch = options.syncSearch;
  return requestJSON("/api/index", {
    method: "POST",
    body: JSON.stringify(body)
  });
}

// -----------------------------------------------------------------------------
// Scan control centre
// -----------------------------------------------------------------------------

/**
 * getScanStatus reads the live progress, pause state and worker breakdown of the
 * running scan, or the most recent finished session when nothing is running.
 *
 * It is deliberately NOT cached: the caller polls it, and a cached snapshot
 * would freeze the progress bar and the worker table.
 */
export async function getScanStatus() {
  return requestJSON("/api/scan/status", { cache: "no-store" });
}

/**
 * getScanWorkers reads only the per-worker state, for a panel that polls more
 * often than the full status.
 */
export async function getScanWorkers() {
  return requestJSON("/api/scan/workers", { cache: "no-store" });
}

/**
 * pauseScan requests a cooperative pause.
 *
 * The response state is usually "pausing": workers finish the file they are on
 * before stopping. The scan only reaches "paused" once none is mid-item, so the
 * UI must show the transition instead of claiming an immediate stop.
 */
export async function pauseScan() {
  return requestJSON("/api/scan/pause", { method: "POST" });
}

/** resumeScan lifts a pause and continues from the same point. */
export async function resumeScan() {
  return requestJSON("/api/scan/resume", { method: "POST" });
}

/** cancelScan ends the scan. It is a different operation from pause. */
export async function cancelScan() {
  return requestJSON("/api/scan/cancel", { method: "POST" });
}

/** getInterruptedScans lists scans that never finished, e.g. after a restart. */
export async function getInterruptedScans() {
  return requestJSON("/api/scan/interrupted", { cache: "no-store" });
}

// -----------------------------------------------------------------------------
// Entity Resolution review queue
// -----------------------------------------------------------------------------

/**
 * getResolutionQueue lists files that entity resolution refused to decide.
 *
 * These are the files that would previously have become works named "01" or
 * after a site watermark. Each item carries the ranked candidates and the
 * itemised evidence behind each score, so an operator can choose instead of
 * retyping a title.
 */
export async function getResolutionQueue(options = {}) {
  const params = new URLSearchParams();
  if (options.reason) params.set("reason", options.reason);
  if (options.limit) params.set("limit", String(options.limit));
  const query = params.toString();
  return requestJSON(`/api/resolution/queue${query ? `?${query}` : ""}`, { cache: "no-store" });
}

/** getResolutionStats reports how many files still need a human, per reason. */
export async function getResolutionStats() {
  return requestJSON("/api/resolution/stats", { cache: "no-store" });
}

/**
 * decideResolution applies an operator decision to a queued file.
 *
 * The decision is recorded for auditability and, when learnAlias is true,
 * promoted into a durable alias so the same ambiguity never appears again.
 *
 * @param {number} itemId        queue item id
 * @param {object} decision      { action, workId?, season?, episode?, newTitle?, learnAlias? }
 */
export async function decideResolution(itemId, decision) {
  return requestJSON(`/api/resolution/queue/${encodeURIComponent(itemId)}/decide`, {
    method: "POST",
    body: JSON.stringify({
      action: decision.action,
      work_id: decision.workId ?? 0,
      season: decision.season ?? 0,
      episode: decision.episode ?? 0,
      new_title: decision.newTitle ?? "",
      learn_alias: Boolean(decision.learnAlias)
    })
  });
}

/**
 * rebuildSearchIndex rebuilds the search index as a projection of PostgreSQL.
 *
 * `reset` starts from the beginning (drop and rebuild); without it the run
 * resumes from the persisted cursor. Nothing here re-reads the filesystem.
 */
export async function rebuildSearchIndex(reset = false) {
  return requestJSON(`/api/search/sync${reset ? "?reset=true" : ""}`, { method: "POST" });
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

export async function getMediaRelated(mediaId, limit = 18) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/related?limit=${encodeURIComponent(limit)}`);
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

export const COPY_BRIDGE_BASE = (import.meta.env.VITE_COPY_BRIDGE_URL || "http://127.0.0.1:32145").replace(/\/$/, "");

export const BRIDGE_OFFLINE_MESSAGE =
  "خدمة NEXORA Copy Bridge غير متصلة على هذا الجهاز (127.0.0.1:32145). يرجى التأكد من تشغيل الخدمة للنسخ عبر USB.";

export function resolveBridgeURL(path = "") {
  if (!path) return COPY_BRIDGE_BASE;
  if (/^https?:\/\//i.test(path)) return path;
  return `${COPY_BRIDGE_BASE}${path.startsWith("/") ? path : `/${path}`}`;
}

export function isBridgeOfflineError(err) {
  const message = err?.message || "";
  return message.includes("Copy Bridge غير متصلة") || message.includes("Failed to fetch") || message.includes("NetworkError") || message.includes("Load failed");
}

export async function getBridgeHealth() {
  try {
    const res = await fetch(resolveBridgeURL("/api/health"), { method: "GET" });
    if (!res.ok) return { ok: false };
    const body = await res.json().catch(() => ({}));
    return { ok: true, ...body };
  } catch {
    return { ok: false };
  }
}

async function requestBridgeJSON(path, options = {}) {
  const url = resolveBridgeURL(path);
  try {
    const res = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(options.headers || {})
      }
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `Bridge HTTP ${res.status}`);
    }
    return await res.json();
  } catch (err) {
    if (isBridgeOfflineError(err)) {
      throw new Error(BRIDGE_OFFLINE_MESSAGE);
    }
    throw err;
  }
}

// USB Transfer API (Communicates locally with NEXORA Copy Bridge on 127.0.0.1:32145)
export async function getTransferDevices() {
  return requestBridgeJSON("/api/transfer/devices");
}

export async function getTransferDeviceApps(deviceId) {
  return requestBridgeJSON(`/api/transfer/device-apps?device_id=${encodeURIComponent(deviceId || "")}`);
}

export async function startDeviceTransfer(payload) {
  return requestBridgeJSON("/api/transfer/copy", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function getTransferJobs() {
  return requestBridgeJSON("/api/transfer/jobs");
}

export async function cancelTransferJob(jobId) {
  return requestBridgeJSON(`/api/transfer/cancel/${encodeURIComponent(jobId)}`, {
    method: "POST"
  });
}

export async function browseTransferPath(deviceId, path = "", deviceType = "", bundleId = "") {
  const params = new URLSearchParams({
    device_id: deviceId || "",
    path: path || ""
  });
  if (deviceType) params.set("device_type", deviceType);
  if (bundleId) params.set("bundle_id", bundleId);
  return requestBridgeJSON(`/api/transfer/browse?${params.toString()}`);
}

export async function createTransferFolder(payload) {
  return requestBridgeJSON("/api/transfer/mkdir", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function ejectTransferDevice(deviceId) {
  return requestBridgeJSON("/api/transfer/eject", {
    method: "POST",
    body: JSON.stringify({ device_id: deviceId })
  });
}

export async function openFileLocation(payload) {
  return requestJSON("/api/system/open-file-location", {
    method: "POST",
    body: JSON.stringify(payload)
  });
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

// resolveTransferStreamURL builds the absolute http(s) URL that the local Copy
// Bridge must fetch over LAN. Using the page origin keeps it pointing at the
// same NEXORA server the browser was opened from (dev proxy or LAN IP alike).
export function resolveTransferStreamURL(path) {
  if (!path) {
    return "";
  }
  if (/^(?:https?:\/\/|data:|blob:)/i.test(path)) {
    return path;
  }
  let origin = API_BASE;
  if (!/^https?:\/\//i.test(origin)) {
    origin =
      typeof window !== "undefined" && window.location
        ? window.location.origin
        : "";
  }
  return `${origin}${path.startsWith("/") ? path : `/${path}`}`;
}
