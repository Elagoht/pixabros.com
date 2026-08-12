import { lazy } from "react";
import { createBrowserRouter } from "react-router-dom";
import { Navigation } from "@/utilities/navigation";

/* Layouts */
const AuthLayout = lazy(() => import("@/app/(auth)/layout"));
const PanelLayout = lazy(() => import("@/app/(panel)/layout"));

/* Pages */
const LoginPage = lazy(() => import("@/app/(auth)/login/page"));
const ForgotPasswordPage = lazy(
  () => import("@/app/(auth)/forgot-password/page"),
);
const MainPage = lazy(() => import("@/app/(panel)/page"));
const ChangePasswordPage = lazy(
  () => import("@/app/(panel)/change-password/page"),
);
const NotFoundPage = lazy(() => import("@/app/not-found"));

// The Go server mounts this SPA under a non-root prefix, so react-router has
// to be told about it or every in-app link would resolve against "/".
export const router = createBrowserRouter(
  [
    {
      element: <AuthLayout />,
      children: [
        {
          element: <LoginPage />,
          path: "/login",
        },
        {
          element: <ForgotPasswordPage />,
          path: "/forgot-password",
        },
      ],
    },
    {
      element: <PanelLayout />,
      children: [
        {
          element: <MainPage />,
          path: "/",
        },
        {
          element: <ChangePasswordPage />,
          path: "/change-password",
        },
      ],
    },
    {
      element: <NotFoundPage />,
      path: "*",
    },
  ],
  { basename: Navigation.basePath || undefined },
);
