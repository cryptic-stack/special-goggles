import { Send } from "lucide-react";
import { FormEvent, useState } from "react";
import { useUiStore } from "../features/ui/store";

type ComposerProps = {
  threadMode?: boolean;
};

export function Composer({ threadMode = false }: ComposerProps) {
  const pushToast = useUiStore((s) => s.pushToast);
  const density = useUiStore((s) => s.density);
  const [text, setText] = useState("");
  const [cwEnabled, setCwEnabled] = useState(false);
  const [cwText, setCwText] = useState("");
  const [visibility, setVisibility] = useState("public");
  const [language, setLanguage] = useState("en");

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!text.trim()) {
      return;
    }
    pushToast("Post queued for publish.");
    setText("");
    setCwText("");
    setCwEnabled(false);
  }

  return (
    <form onSubmit={onSubmit} className={`rounded-xl border border-ui-line bg-ui-panel ${density === "comfortable" ? "p-5" : "p-4"}`}>
      <div className="mb-2 flex items-center justify-between">
        <h3 className="font-medium">{threadMode ? "Thread Composer" : "Composer"}</h3>
        <label className="flex items-center gap-2 text-xs text-ui-muted">
          <input type="checkbox" checked={cwEnabled} onChange={(event) => setCwEnabled(event.target.checked)} />
          CW
        </label>
      </div>
      {cwEnabled ? (
        <input
          value={cwText}
          onChange={(event) => setCwText(event.target.value)}
          placeholder="Content warning"
          className="mb-2 w-full rounded-lg border border-ui-line bg-ui-surface px-3 py-2 text-sm"
        />
      ) : null}
      <textarea
        value={text}
        onChange={(event) => setText(event.target.value)}
        placeholder={threadMode ? "Write a thread reply..." : "What's happening?"}
        className="min-h-28 w-full rounded-lg border border-ui-line bg-ui-surface px-3 py-2 text-sm"
      />
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <label className="text-xs text-ui-muted">
          Visibility
          <select
            value={visibility}
            onChange={(event) => setVisibility(event.target.value)}
            className="ml-2 rounded-md border border-ui-line bg-ui-surface px-2 py-1"
          >
            <option value="public">Public</option>
            <option value="unlisted">Unlisted</option>
            <option value="followers">Followers</option>
            <option value="direct">Direct</option>
          </select>
        </label>
        <label className="text-xs text-ui-muted">
          Language
          <input
            value={language}
            onChange={(event) => setLanguage(event.target.value)}
            className="ml-2 w-16 rounded-md border border-ui-line bg-ui-surface px-2 py-1"
          />
        </label>
        <button
          type="submit"
          className="ml-auto inline-flex items-center gap-2 rounded-md bg-ui-accent px-3 py-2 text-sm font-semibold text-ui-bg"
        >
          <Send size={14} />
          Post
        </button>
      </div>
    </form>
  );
}
