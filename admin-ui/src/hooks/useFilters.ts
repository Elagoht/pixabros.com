import { useSearchParams } from "react-router-dom";

const RESERVED_KEYS = ["page", "pageSize", "sortBy", "sortDir"];

export const useFilters = () => {
  const [searchParams] = useSearchParams();

  const filters: Record<string, string> = {};
  for (const [key, value] of searchParams.entries()) {
    if (!RESERVED_KEYS.includes(key) && value) {
      filters[key] = value;
    }
  }

  return filters;
};
