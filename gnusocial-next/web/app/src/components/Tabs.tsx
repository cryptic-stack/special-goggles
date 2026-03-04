type TabsProps = {
  tabs: string[];
  active: string;
  onChange: (tab: string) => void;
};

export function Tabs({ tabs, active, onChange }: TabsProps) {
  return (
    <div className="flex gap-1 rounded-lg border border-ui-line bg-ui-panel p-1">
      {tabs.map((tab) => (
        <button
          key={tab}
          type="button"
          onClick={() => onChange(tab)}
          className={`rounded-md px-3 py-1 text-sm ${
            active === tab ? "bg-ui-accent text-ui-bg" : "text-ui-muted hover:text-ui-text"
          }`}
        >
          {tab}
        </button>
      ))}
    </div>
  );
}
