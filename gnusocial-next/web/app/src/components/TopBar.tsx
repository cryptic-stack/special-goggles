import type { ReactNode } from "react";

type TopBarProps = {
  title: string;
  controls?: ReactNode;
};

export function TopBar({ title, controls }: TopBarProps) {
  return (
    <header className="sticky top-0 z-20 border-b border-ui-line bg-ui-surface/95 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-feed items-center justify-between gap-3 px-4">
        <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
        <div className="flex items-center gap-2">{controls}</div>
      </div>
    </header>
  );
}
