import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { Environment } from "@/utilities/environment";

export const usePageParams = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = Number(searchParams.get("page") || 1);
  const pageSize = Number(searchParams.get("pageSize") || Environment.pageSize);

  const setPage = useCallback(
    (p: number) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("page", String(p));
        return next;
      });
    },
    [setSearchParams],
  );

  const setPageSize = useCallback(
    (size: number) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("pageSize", String(size));
        next.set("page", "1");
        return next;
      });
    },
    [setSearchParams],
  );

  return { page, pageSize, setPage, setPageSize };
};
