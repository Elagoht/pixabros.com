import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";

export const useSortParams = (fieldMap: Record<string, string>) => {
  const [searchParams, setSearchParams] = useSearchParams();

  const sortBy = searchParams.get("sortBy") ?? undefined;
  const sortDir = (searchParams.get("sortDir") as "asc" | "desc") || undefined;

  const ordering = sortBy
    ? sortDir === "asc"
      ? (fieldMap[sortBy] ?? sortBy)
      : `-${fieldMap[sortBy] ?? sortBy}`
    : undefined;

  const setSort = useCallback(
    (columnId: string, dir: "asc" | "desc") => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("sortBy", columnId);
        next.set("sortDir", dir);
        next.set("page", "1");
        return next;
      });
    },
    [setSearchParams],
  );

  return { sortBy, sortDir, ordering, setSort };
};
