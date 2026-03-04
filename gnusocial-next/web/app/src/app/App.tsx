import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { useEffect } from "react";
import { useUiStore } from "../features/ui/store";
import { queryClient } from "../lib/queryClient";
import { router } from "./router";

function UiBindings() {
  const density = useUiStore((s) => s.density);
  const reducedMotion = useUiStore((s) => s.reducedMotion);

  useEffect(() => {
    document.documentElement.dataset.density = density;
  }, [density]);

  useEffect(() => {
    document.documentElement.dataset.reducedMotion = String(reducedMotion);
  }, [reducedMotion]);

  return null;
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <UiBindings />
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
