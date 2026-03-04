import type { Post } from "../lib/types";

type MediaGridProps = {
  media: Post["media"];
  density: "comfortable" | "default" | "compact";
};

export function MediaGrid({ media, density }: MediaGridProps) {
  if (!media.length) {
    return null;
  }

  const heightClass =
    density === "comfortable" ? "aspect-[16/9]" : density === "compact" ? "aspect-[4/3]" : "aspect-[3/2]";

  return (
    <div className="mt-3 grid gap-2">
      {media.map((item) => (
        <figure key={item.id} className={`relative overflow-hidden rounded-xl border border-ui-line ${heightClass}`}>
          <img
            src={item.previewUrl}
            alt={item.alt}
            loading="lazy"
            className="absolute inset-0 h-full w-full object-cover"
            width={item.width}
            height={item.height}
          />
        </figure>
      ))}
    </div>
  );
}
