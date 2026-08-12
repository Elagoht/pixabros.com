export const parseBusinessArea = (raw: string | null | undefined): string[] => {
  if (!raw) {
    return [];
  }
  try {
    const arr = JSON.parse(raw);
    return Array.isArray(arr) ? arr : [];
  } catch {
    return [];
  }
};
