export function SkeletonPost() {
  return (
    <div className="animate-pulse rounded-xl border border-ui-line bg-ui-panel p-4">
      <div className="flex items-center gap-3">
        <div className="h-10 w-10 rounded-full bg-ui-line" />
        <div className="space-y-2">
          <div className="h-3 w-32 rounded bg-ui-line" />
          <div className="h-2 w-24 rounded bg-ui-line" />
        </div>
      </div>
      <div className="mt-4 h-3 w-full rounded bg-ui-line" />
      <div className="mt-2 h-3 w-4/5 rounded bg-ui-line" />
      <div className="mt-4 aspect-[16/9] rounded-lg bg-ui-line" />
    </div>
  );
}

export function SkeletonList({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: count }, (_, i) => (
        <SkeletonPost key={`skeleton-${i}`} />
      ))}
    </div>
  );
}
