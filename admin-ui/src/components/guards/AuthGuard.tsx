import type { FC, ReactNode } from "react";
import { useEffect } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { Loading } from "@/components/ui";
import { useAuthStore } from "@/lib/stores/auth";

interface AuthGuardProps {
  children: ReactNode;
}

const AuthGuard: FC<AuthGuardProps> = ({ children }) => {
  const { isAuthenticated, isLoading, user, checkAuth } = useAuthStore();
  const location = useLocation();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  if (!isAuthenticated && isLoading) {
    return <Loading />;
  }

  if (!isAuthenticated) {
    const currentPath = location.pathname + location.search + location.hash;
    return (
      <Navigate to={`/login?next=${encodeURIComponent(currentPath)}`} replace />
    );
  }

  if (user?.first_login && location.pathname !== "/reset-password") {
    return <Navigate to="/reset-password" replace />;
  }

  return <>{children}</>;
};

export default AuthGuard;
