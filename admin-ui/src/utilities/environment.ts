export const Environment = {
  apiBase: import.meta.env.VITE_API_BASE ?? "http://localhost:3000",
  mediaBase: import.meta.env.VITE_MEDIA_BASE ?? "http://localhost:3001",
  pageSize: Number(import.meta.env.VITE_PAGE_SIZE) || 12,
};
