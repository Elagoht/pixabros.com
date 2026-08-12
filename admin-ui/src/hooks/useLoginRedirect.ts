import { useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

export const useLoginRedirect = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const next = searchParams.get("next");

  return useCallback(
    () => navigate(next || "/", { replace: true }),
    [navigate, next],
  );
};
