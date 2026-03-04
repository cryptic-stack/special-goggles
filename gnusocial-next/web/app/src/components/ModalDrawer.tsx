import type { ReactNode } from "react";

type LayerProps = {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
};

export function Modal({ open, title, onClose, children }: LayerProps) {
  if (!open) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-50 bg-black/50 p-4">
      <div className="mx-auto mt-20 max-w-xl rounded-xl border border-ui-line bg-ui-surface p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="rounded-md border border-ui-line px-2 py-1 text-xs">
            Close
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function Drawer({ open, title, onClose, children }: LayerProps) {
  if (!open) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-50 bg-black/50">
      <div className="ml-auto h-full w-full max-w-md border-l border-ui-line bg-ui-surface p-4 shadow-2xl">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="rounded-md border border-ui-line px-2 py-1 text-xs">
            Close
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
