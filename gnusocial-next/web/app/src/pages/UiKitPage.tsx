import { useEffect, useState } from "react";
import { Composer } from "../components/Composer";
import { EmptyState } from "../components/EmptyState";
import { FiltersBar } from "../components/FiltersBar";
import { Modal } from "../components/ModalDrawer";
import { PostCard } from "../components/PostCard";
import { SkeletonPost } from "../components/Skeletons";
import { Tabs } from "../components/Tabs";
import { TopBar } from "../components/TopBar";
import { fetchTimeline } from "../lib/mockData";
import type { Post } from "../lib/types";

export function UiKitPage() {
  const [tab, setTab] = useState("Components");
  const [modalOpen, setModalOpen] = useState(false);
  const [post, setPost] = useState<Post | null>(null);
  const [chips, setChips] = useState([
    { id: "a", label: "Media", active: false },
    { id: "b", label: "Replies", active: false }
  ]);

  useEffect(() => {
    void fetchTimeline("home", null, 1).then((rows) => setPost(rows.items[0]));
  }, []);

  return (
    <>
      <TopBar title="UI Kit" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Tokens</h2>
          <p className="mt-2 text-sm text-ui-muted">
            Spacing 4/8/12/16/24/32, type scale 12/14/16/18/20/24/32, center width 680-760.
          </p>
        </section>
        <Tabs tabs={["Components", "Layouts", "A11y"]} active={tab} onChange={setTab} />
        <section className="space-y-3 rounded-xl border border-ui-line bg-ui-panel p-4">
          <h3 className="text-sm font-semibold">Component Gallery</h3>
          <FiltersBar
            chips={chips}
            onToggleChip={(id) =>
              setChips((current) => current.map((chip) => (chip.id === id ? { ...chip, active: !chip.active } : chip)))
            }
            language=""
            onLanguageChange={() => undefined}
          />
          {post ? <PostCard post={post} /> : <SkeletonPost />}
          <Composer />
          <EmptyState title="Empty state sample" description="Use this component to teach the next action." />
          <button type="button" onClick={() => setModalOpen(true)} className="rounded-md border border-ui-line px-3 py-2 text-sm">
            Open modal
          </button>
        </section>
      </div>
      <Modal open={modalOpen} title="Modal Sample" onClose={() => setModalOpen(false)}>
        <p className="text-sm text-ui-muted">Modal and drawer primitives are available for settings and moderation flows.</p>
      </Modal>
    </>
  );
}
