import { useEffect } from "react";
import { useUiStore } from "../features/ui/store";

export function Toasts() {
  const toasts = useUiStore((s) => s.toasts);
  const removeToast = useUiStore((s) => s.removeToast);

  useEffect(() => {
    if (!toasts.length) {
      return;
    }
    const timers = toasts.map((toast) => setTimeout(() => removeToast(toast.id), 2800));
    return () => timers.forEach((timer) => clearTimeout(timer));
  }, [toasts, removeToast]);

  return (
    <div className="pointer-events-none fixed bottom-4 right-4 z-50 space-y-2">
      {toasts.map((toast) => (
        <div key={toast.id} className="pointer-events-auto rounded-md border border-ui-line bg-ui-panel px-3 py-2 text-sm">
          {toast.message}
        </div>
      ))}
    </div>
  );
}
