import { SlidersHorizontal } from "lucide-react";

type FiltersBarProps = {
  chips: Array<{ id: string; label: string; active: boolean }>;
  onToggleChip: (id: string) => void;
  language: string;
  onLanguageChange: (value: string) => void;
};

export function FiltersBar({ chips, onToggleChip, language, onLanguageChange }: FiltersBarProps) {
  return (
    <section className="border-b border-ui-line bg-ui-panel px-4 py-2">
      <div className="mx-auto flex max-w-feed flex-wrap items-center gap-2">
        <div className="mr-1 flex items-center gap-1 text-xs text-ui-muted">
          <SlidersHorizontal size={14} />
          <span>Filters</span>
        </div>
        {chips.map((chip) => (
          <button
            key={chip.id}
            type="button"
            onClick={() => onToggleChip(chip.id)}
            className={`rounded-full border px-3 py-1 text-xs ${
              chip.active ? "border-ui-accent bg-ui-accent/20 text-ui-text" : "border-ui-line text-ui-muted"
            }`}
          >
            {chip.label}
          </button>
        ))}
        <label className="ml-auto flex items-center gap-2 text-xs text-ui-muted">
          <span>Language</span>
          <select
            value={language}
            onChange={(event) => onLanguageChange(event.target.value)}
            className="rounded-md border border-ui-line bg-ui-surface px-2 py-1 text-xs"
          >
            <option value="">All</option>
            <option value="en">English</option>
            <option value="en-US">English (US)</option>
            <option value="es">Spanish</option>
          </select>
        </label>
      </div>
    </section>
  );
}
