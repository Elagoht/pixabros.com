/* eslint-disable react-refresh/only-export-components */
import { lazy } from "react";
import { createBrowserRouter } from "react-router-dom";

/* Layouts */
const AuthLayout = lazy(() => import("@/app/(auth)/layout"));
const PanelLayout = lazy(() => import("@/app/(panel)/layout"));

/* Pages */
const LoginPage = lazy(() => import("@/app/(auth)/login/page"));
const ForgotPasswordPage = lazy(
  () => import("@/app/(auth)/forgot-password/page"),
);
const ResetPasswordPage = lazy(
  () => import("@/app/(auth)/reset-password/page"),
);
const MainPage = lazy(() => import("@/app/(panel)/page"));
const UsersPage = lazy(() => import("@/app/(panel)/users/page"));
const UserCreatePage = lazy(() => import("@/app/(panel)/users/new/page"));
const UserDetailPage = lazy(() => import("@/app/(panel)/users/[id]/page"));
const NotFoundPage = lazy(() => import("@/app/not-found"));

export const router = createBrowserRouter([
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
    element: <ResetPasswordPage />,
    path: "/reset-password",
  },
  {
    element: <PanelLayout />,
    children: [
      {
        element: <MainPage />,
        path: "/",
      },
      {
        element: <UsersPage />,
        path: "/definitions/users",
      },
      {
        element: <UserCreatePage />,
        path: "/definitions/users/new",
      },
      {
        element: <UserDetailPage />,
        path: "/definitions/users/:id",
      },
    ],
  },
  {
    element: <NotFoundPage />,
    path: "*",
  },
]);
