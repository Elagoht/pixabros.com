import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@/design/index.css";
import "@/lib/stores/i18n";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { Toaster } from "sonner";
import { router } from "@/lib/routes";
import { queryClient } from "./lib/query/client";

const root = document.getElementById("root");
if (!root) {
  throw new Error("Root element not found");
}
createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />

      <Toaster richColors position="bottom-right" />
    </QueryClientProvider>
  </StrictMode>,
);
