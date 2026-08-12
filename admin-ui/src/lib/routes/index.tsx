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
const GamesPage = lazy(() => import("@/app/(panel)/games/page"));
const GameCreatePage = lazy(() => import("@/app/(panel)/games/new/page"));
const GameEditPage = lazy(() => import("@/app/(panel)/games/[id]/page"));
const MembersPage = lazy(() => import("@/app/(panel)/members/page"));
const MemberCreatePage = lazy(() => import("@/app/(panel)/members/new/page"));
const MemberEditPage = lazy(() => import("@/app/(panel)/members/[id]/page"));
const AwardsPage = lazy(() => import("@/app/(panel)/awards/page"));
const AwardCreatePage = lazy(() => import("@/app/(panel)/awards/new/page"));
const AwardEditPage = lazy(() => import("@/app/(panel)/awards/[id]/page"));
const DevlogPage = lazy(() => import("@/app/(panel)/devlog/page"));
const DevlogCreatePage = lazy(() => import("@/app/(panel)/devlog/new/page"));
const DevlogEditPage = lazy(() => import("@/app/(panel)/devlog/[id]/page"));
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
        {
          element: <GamesPage />,
          path: "/games",
        },
        // Registered above /games/:id so the static segment is matched first.
        {
          element: <GameCreatePage />,
          path: "/games/new",
        },
        {
          element: <GameEditPage />,
          path: "/games/:id",
        },
        {
          element: <MembersPage />,
          path: "/members",
        },
        // Above /members/:id so the static segment matches first.
        {
          element: <MemberCreatePage />,
          path: "/members/new",
        },
        {
          element: <MemberEditPage />,
          path: "/members/:id",
        },
        {
          element: <AwardsPage />,
          path: "/awards",
        },
        // Above /awards/:id so the static segment matches first.
        {
          element: <AwardCreatePage />,
          path: "/awards/new",
        },
        {
          element: <AwardEditPage />,
          path: "/awards/:id",
        },
        {
          element: <DevlogPage />,
          path: "/devlog",
        },
        // Above /devlog/:id so the static segment matches first.
        {
          element: <DevlogCreatePage />,
          path: "/devlog/new",
        },
        {
          element: <DevlogEditPage />,
          path: "/devlog/:id",
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
