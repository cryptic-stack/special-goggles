import { Tabs } from "../components/Tabs";
import { TopBar } from "../components/TopBar";
import { useState } from "react";

export function ProfilePage() {
  const [tab, setTab] = useState("Posts");
  return (
    <>
      <TopBar title="Profile" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <div className="flex items-center gap-3">
            <img src="https://i.pravatar.cc/80?img=10" alt="me avatar" className="h-14 w-14 rounded-full border border-ui-line" />
            <div>
              <p className="font-semibold">Admin User</p>
              <p className="text-sm text-ui-muted">@admin@localhost</p>
            </div>
          </div>
          <div className="mt-3 flex gap-2">
            <button className="rounded-md border border-ui-line px-3 py-1 text-xs">Follow</button>
            <button className="rounded-md border border-ui-line px-3 py-1 text-xs">Mute</button>
            <button className="rounded-md border border-ui-line px-3 py-1 text-xs">Block</button>
            <button className="rounded-md border border-ui-line px-3 py-1 text-xs">Add to list</button>
          </div>
        </section>
        <Tabs tabs={["Posts", "Replies", "Media", "About"]} active={tab} onChange={setTab} />
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4 text-sm text-ui-muted">
          {tab === "About"
            ? "Verified links and profile fields render here."
            : `${tab} timeline placeholder. Integrate with profile endpoints in the next pass.`}
        </section>
      </div>
    </>
  );
}
