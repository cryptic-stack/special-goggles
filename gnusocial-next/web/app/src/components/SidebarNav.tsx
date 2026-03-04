import { Bell, Bookmark, Compass, Flag, Home, List, Search, Settings, Shield, User } from "lucide-react";
import type { ComponentType } from "react";
import { NavLink } from "react-router-dom";
import { useUiStore } from "../features/ui/store";

type NavItem = {
  to: string;
  label: string;
  icon: ComponentType<{ size?: string | number }>;
  role?: "moderator" | "admin";
  hidden?: boolean;
};

const baseNav: NavItem[] = [
  { to: "/home", label: "Home", icon: Home },
  { to: "/following", label: "Following", icon: User },
  { to: "/local", label: "Local", icon: Compass },
  { to: "/federated", label: "Federated", icon: Compass },
  { to: "/explore", label: "Explore", icon: Compass },
  { to: "/search", label: "Search", icon: Search },
  { to: "/notifications", label: "Notifications", icon: Bell },
  { to: "/messages", label: "Messages", icon: Bell },
  { to: "/lists", label: "Lists", icon: List },
  { to: "/bookmarks", label: "Bookmarks", icon: Bookmark },
  { to: "/profile", label: "Profile", icon: User },
  { to: "/settings", label: "Settings", icon: Settings },
  { to: "/moderation", label: "Moderation", icon: Flag, role: "moderator" },
  { to: "/admin", label: "Admin", icon: Shield, role: "admin" }
];

export function SidebarNav() {
  const role = useUiStore((s) => s.role);
  const messagesEnabled = useUiStore((s) => s.messagesEnabled);
  const density = useUiStore((s) => s.density);
  const setDensity = useUiStore((s) => s.setDensity);
  const setCommandOpen = useUiStore((s) => s.setCommandOpen);

  const navItems = baseNav.filter((item) => {
    if (item.to === "/messages" && !messagesEnabled) {
      return false;
    }
    if (!item.role) {
      return true;
    }
    if (item.role === "moderator") {
      return role === "moderator" || role === "admin";
    }
    return role === "admin";
  });

  return (
    <aside className="sticky top-0 hidden h-screen border-r border-ui-line bg-ui-surface p-4 md:block">
      <div className="rounded-xl border border-ui-line bg-ui-panel p-3">
        <p className="text-xs uppercase tracking-wide text-ui-muted">Identity</p>
        <p className="mt-1 text-sm font-semibold">admin@localhost</p>
        <div className="mt-2">
          <label className="text-xs text-ui-muted">Instance</label>
          <select className="mt-1 w-full rounded-md border border-ui-line bg-ui-surface px-2 py-1 text-sm">
            <option>localhost</option>
            <option>community.example</option>
          </select>
        </div>
      </div>

      <button
        type="button"
        onClick={() => setCommandOpen(true)}
        className="mt-3 w-full rounded-lg border border-ui-line bg-ui-panel px-3 py-2 text-left text-sm text-ui-muted"
      >
        Quick Search
        <span className="float-right rounded border border-ui-line px-1 text-xs">Ctrl+K</span>
      </button>

      <nav className="mt-3 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center gap-2 rounded-lg px-3 py-2 text-sm ${
                  isActive ? "bg-ui-accent text-ui-bg" : "text-ui-text hover:bg-ui-panel"
                }`
              }
            >
              <Icon size={16} />
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>

      <div className="mt-4 rounded-xl border border-ui-line bg-ui-panel p-3">
        <p className="text-xs uppercase tracking-wide text-ui-muted">Density</p>
        <div className="mt-2 grid grid-cols-3 gap-1">
          {(["comfortable", "default", "compact"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => setDensity(mode)}
              className={`rounded-md px-2 py-1 text-xs ${
                density === mode ? "bg-ui-accent text-ui-bg" : "bg-ui-surface text-ui-muted"
              }`}
            >
              {mode}
            </button>
          ))}
        </div>
      </div>
    </aside>
  );
}
