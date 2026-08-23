const API_BASE = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");

async function requestJSON(path, options = {}) {
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

  return data;
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

export async function getMediaDetail(mediaId) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}`);
}

export async function getMediaMetadataSnapshot(mediaId, locale = "ar-SA") {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/metadata/raw?locale=${encodeURIComponent(locale)}`);
}

export async function getMediaSeasonMetadata(mediaId, locale = "ar-SA") {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/metadata/seasons?locale=${encodeURIComponent(locale)}`);
}

export async function enrichMedia(mediaId) {
  return requestJSON(`/api/media/${encodeURIComponent(mediaId)}/enrich`, {
    method: "POST"
  });
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

export async function getTMDBModules() {
  return requestJSON("/api/tmdb/modules");
}

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

export function resolveAPIURL(path) {
  if (!path) {
    return "";
  }
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  return `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
}
