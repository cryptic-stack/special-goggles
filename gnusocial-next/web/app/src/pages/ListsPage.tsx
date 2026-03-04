import { FormEvent, useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

type ListItem = { id: string; name: string; accounts: string[] };

export function ListsPage() {
  const pushToast = useUiStore((s) => s.pushToast);
  const [lists, setLists] = useState<ListItem[]>([
    { id: "l1", name: "Infra Team", accounts: ["@devonpike", "@arinorth"] }
  ]);
  const [selected, setSelected] = useState("l1");
  const [accountInput, setAccountInput] = useState("");

  function createList(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const name = String(form.get("name") ?? "").trim();
    if (!name) {
      return;
    }
    const id = `l-${Date.now()}`;
    setLists((prev) => [...prev, { id, name, accounts: [] }]);
    setSelected(id);
    pushToast("List created.");
    event.currentTarget.reset();
  }

  function addAccount() {
    if (!accountInput.trim()) {
      return;
    }
    setLists((prev) =>
      prev.map((list) =>
        list.id === selected ? { ...list, accounts: [...list.accounts, accountInput.trim()] } : list
      )
    );
    setAccountInput("");
    pushToast("Account added to list.");
  }

  const activeList = lists.find((l) => l.id === selected);

  return (
    <>
      <TopBar title="Lists" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <form onSubmit={createList} className="rounded-xl border border-ui-line bg-ui-panel p-3">
          <div className="flex gap-2">
            <input name="name" placeholder="Create list" className="flex-1 rounded-md border border-ui-line bg-ui-surface px-3 py-2 text-sm" />
            <button type="submit" className="rounded-md border border-ui-line px-3 py-2 text-sm">
              Create
            </button>
          </div>
        </form>
        {lists.length === 0 ? (
          <EmptyState title="No lists yet" description="Create lists to curate timeline slices." />
        ) : (
          <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
            <div className="mb-3 flex flex-wrap gap-2">
              {lists.map((list) => (
                <button
                  key={list.id}
                  type="button"
                  onClick={() => setSelected(list.id)}
                  className={`rounded-md px-3 py-1 text-sm ${
                    selected === list.id ? "bg-ui-accent text-ui-bg" : "border border-ui-line"
                  }`}
                >
                  {list.name}
                </button>
              ))}
            </div>
            {activeList ? (
              <>
                <h2 className="text-sm font-semibold">{activeList.name}</h2>
                <div className="mt-2 flex gap-2">
                  <input
                    value={accountInput}
                    onChange={(event) => setAccountInput(event.target.value)}
                    placeholder="Add account handle"
                    className="flex-1 rounded-md border border-ui-line bg-ui-surface px-3 py-2 text-sm"
                  />
                  <button type="button" onClick={addAccount} className="rounded-md border border-ui-line px-3 py-2 text-sm">
                    Add
                  </button>
                </div>
                <ul className="mt-3 space-y-2 text-sm">
                  {activeList.accounts.map((account) => (
                    <li key={account} className="flex items-center justify-between rounded-md border border-ui-line px-3 py-2">
                      <span>{account}</span>
                      <button
                        type="button"
                        className="text-xs text-ui-muted"
                        onClick={() =>
                          setLists((prev) =>
                            prev.map((list) =>
                              list.id === selected
                                ? { ...list, accounts: list.accounts.filter((a) => a !== account) }
                                : list
                            )
                          )
                        }
                      >
                        Remove
                      </button>
                    </li>
                  ))}
                </ul>
              </>
            ) : null}
          </section>
        )}
      </div>
    </>
  );
}
