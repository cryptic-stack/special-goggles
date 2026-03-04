import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

export function MessagesPage() {
  const enabled = useUiStore((s) => s.messagesEnabled);
  return (
    <>
      <TopBar title="Messages" />
      <div className="mx-auto max-w-feed p-4">
        {!enabled ? (
          <EmptyState
            title="Messages disabled"
            description="Enable direct messages in settings to access this route."
          />
        ) : (
          <EmptyState
            title="No messages yet"
            description="Direct messaging is optional and can be feature-flagged per instance."
          />
        )}
      </div>
    </>
  );
}
