import type { FC, ReactNode } from "react";
import { useEffect } from "react";
import { Navigate, useSearchParams } from "react-router-dom";
import { Loading } from "@/components/ui";
import { useAuthStore } from "@/lib/stores/auth";

interface GuestGuardProps {
  children: ReactNode;
}

const GuestGuard: FC<GuestGuardProps> = ({ children }) => {
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  if (isLoading) {
    return <Loading />;
  }

  if (isAuthenticated) {
    const next = searchParams.get("next");
    return <Navigate to={next || "/"} replace />;
  }

  return <>{children}</>;
};

export default GuestGuard;
