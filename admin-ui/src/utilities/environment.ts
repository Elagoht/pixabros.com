// The Go server serves both the admin SPA and /api, so same-origin (an empty
// base) is correct in production; the Vite dev server proxies /api and /media
// to the backend so the same empty base works in development.
export const Environment = {
  apiBase: import.meta.env.VITE_API_BASE ?? "",
  mediaBase: import.meta.env.VITE_MEDIA_BASE ?? "",
  pageSize: Number(import.meta.env.VITE_PAGE_SIZE) || 12,
};
