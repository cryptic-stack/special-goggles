import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  createPost,
  createUser,
  fetchHomeTimeline,
  fetchPublicTimeline,
  follow,
  hidePost,
  mute,
  unfollow,
  unmute
} from "./api";
import type { Post } from "./types";

const PAGE_SIZE = 20;

type ThemeName = "gnusocial-dark" | "graphite" | "midnight";
type FeedMode = "public" | "home";

const THEMES: ThemeName[] = ["gnusocial-dark", "graphite", "midnight"];

export default function App() {
  const [feedMode, setFeedMode] = useState<FeedMode>("public");
  const [posts, setPosts] = useState<Post[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
  const [status, setStatus] = useState<string>("");
  const [composerText, setComposerText] = useState("");
  const [search, setSearch] = useState("");
  const [currentUserId, setCurrentUserId] = useState(localStorage.getItem("currentUserId") ?? "");
  const [currentUsername, setCurrentUsername] = useState(localStorage.getItem("currentUsername") ?? "");
  const [theme, setTheme] = useState<ThemeName>("gnusocial-dark");
  const [registering, setRegistering] = useState(false);

  const sentinelRef = useRef<HTMLDivElement | null>(null);

  const themeStorageKey = currentUserId ? `theme:${currentUserId}` : "theme:guest";

  useEffect(() => {
    const saved = localStorage.getItem(themeStorageKey) as ThemeName | null;
    if (saved && THEMES.includes(saved)) {
      setTheme(saved);
    }
  }, [themeStorageKey]);

  useEffect(() => {
    localStorage.setItem(themeStorageKey, theme);
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme, themeStorageKey]);

  const loadPosts = useCallback(
    async (reset: boolean) => {
      if (loading) {
        return;
      }

      if (feedMode === "home" && !currentUserId) {
        setError("Set a current user ID to load the home timeline.");
        return;
      }

      setLoading(true);
      setError("");
      try {
        const nextOffset = reset ? 0 : offset;
        const page =
          feedMode === "public"
            ? await fetchPublicTimeline(nextOffset, PAGE_SIZE, currentUserId || undefined)
            : await fetchHomeTimeline(currentUserId, nextOffset, PAGE_SIZE);

        setHasMore(page.length === PAGE_SIZE);
        setOffset(nextOffset + page.length);
        setPosts((prev) => {
          const merged = reset ? page : [...prev, ...page];
          const unique = new Map<string, Post>();
          for (const post of merged) {
            unique.set(post.id, post);
          }
          return [...unique.values()];
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load timeline.");
      } finally {
        setLoading(false);
      }
    },
    [feedMode, currentUserId, offset, loading]
  );

  useEffect(() => {
    setOffset(0);
    setHasMore(true);
    setPosts([]);
    void loadPosts(true);
  }, [feedMode, currentUserId, loadPosts]);

  useEffect(() => {
    const node = sentinelRef.current;
    if (!node) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.some((entry) => entry.isIntersecting);
        if (visible && hasMore && !loading) {
          void loadPosts(false);
        }
      },
      { threshold: 0.2 }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [hasMore, loading, loadPosts]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!loading) {
        void loadPosts(true);
      }
    }, 30000);

    return () => window.clearInterval(timer);
  }, [loadPosts, loading]);

  async function onCreatePost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!currentUserId) {
      setError("Set a current user ID before posting.");
      return;
    }
    if (!composerText.trim()) {
      return;
    }

    try {
      const post = await createPost({
        authorId: currentUserId,
        content: composerText.trim(),
        visibility: "public"
      });
      setComposerText("");
      setPosts((prev) => [post, ...prev]);
      setStatus("Post published.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to create post.");
    }
  }

  async function onRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const username = String(form.get("username") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "").trim();
    const displayName = String(form.get("displayName") ?? "").trim();
    const bio = String(form.get("bio") ?? "").trim();

    if (!username || !email || !password || !displayName) {
      setError("Registration fields are required.");
      return;
    }

    try {
      setRegistering(true);
      const user = await createUser({ username, email, password, displayName, bio });
      setCurrentUserId(user.id);
      setCurrentUsername(user.username);
      localStorage.setItem("currentUserId", user.id);
      localStorage.setItem("currentUsername", user.username);
      setStatus(`Signed in as @${user.username}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed.");
    } finally {
      setRegistering(false);
    }
  }

  async function runAction(action: "follow" | "unfollow" | "mute" | "unmute" | "hide", target: Post) {
    if (!currentUserId) {
      setError("Set a current user ID before using moderation actions.");
      return;
    }

    try {
      if (action === "follow") {
        await follow(currentUserId, target.authorId);
      } else if (action === "unfollow") {
        await unfollow(currentUserId, target.authorId);
      } else if (action === "mute") {
        await mute(currentUserId, target.authorId);
      } else if (action === "unmute") {
        await unmute(currentUserId, target.authorId);
      } else if (action === "hide") {
        await hidePost(currentUserId, target.id);
        setPosts((prev) => prev.filter((post) => post.id !== target.id));
      }
      setStatus(`${action} completed.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action}.`);
    }
  }

  const filteredPosts = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) {
      return posts;
    }
    return posts.filter((post) => post.content.toLowerCase().includes(query) || post.authorId.toLowerCase().includes(query));
  }, [posts, search]);

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>gnusocial-next</h1>
        <p className="subtle">Modern GNU Social platform</p>
        <nav className="menu">
          <button className={feedMode === "public" ? "active" : ""} onClick={() => setFeedMode("public")}>
            Public Timeline
          </button>
          <button className={feedMode === "home" ? "active" : ""} onClick={() => setFeedMode("home")}>
            Home Timeline
          </button>
          <button onClick={() => setStatus("Notifications stream is available through timeline polling in this milestone.")}>
            Notifications
          </button>
          <button onClick={() => setStatus("Profile editing lands in the next milestone.")}>Profile</button>
          <button onClick={() => setStatus("Moderation actions are available on each post card.")}>Moderation</button>
        </nav>
        <div className="theme-control">
          <label htmlFor="theme">Theme</label>
          <select id="theme" value={theme} onChange={(e) => setTheme(e.target.value as ThemeName)}>
            {THEMES.map((themeName) => (
              <option key={themeName} value={themeName}>
                {themeName}
              </option>
            ))}
          </select>
        </div>
      </aside>

      <main className="feed-panel">
        <section className="session-card">
          <div className="session-row">
            <label>
              Current User ID
              <input
                value={currentUserId}
                onChange={(e) => {
                  const next = e.target.value.trim();
                  setCurrentUserId(next);
                  localStorage.setItem("currentUserId", next);
                }}
                placeholder="UUID"
              />
            </label>
            <label>
              Username
              <input
                value={currentUsername}
                onChange={(e) => {
                  const next = e.target.value.trim();
                  setCurrentUsername(next);
                  localStorage.setItem("currentUsername", next);
                }}
                placeholder="@username"
              />
            </label>
          </div>
          <form className="register-form" onSubmit={onRegister}>
            <input name="username" placeholder="username" />
            <input name="displayName" placeholder="display name" />
            <input name="email" placeholder="email" />
            <input name="password" type="password" placeholder="password" />
            <input name="bio" placeholder="bio (optional)" />
            <button disabled={registering} type="submit">
              {registering ? "Creating..." : "Create User"}
            </button>
          </form>
        </section>

        <section className="composer-card">
          <form onSubmit={onCreatePost}>
            <textarea
              value={composerText}
              onChange={(e) => setComposerText(e.target.value)}
              placeholder="What's happening in your community?"
            />
            <div className="composer-footer">
              <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search loaded posts..." />
              <button type="submit">Publish</button>
            </div>
          </form>
        </section>

        {error ? <p className="banner error">{error}</p> : null}
        {status ? <p className="banner ok">{status}</p> : null}

        <section className="timeline">
          {filteredPosts.map((post) => (
            <article key={post.id} className="post">
              <header>
                <strong>{post.authorId}</strong>
                <span>{new Date(post.createdAt).toLocaleString()}</span>
              </header>
              <p>{post.content}</p>
              <footer>
                <button onClick={() => runAction("follow", post)}>Follow</button>
                <button onClick={() => runAction("unfollow", post)}>Unfollow</button>
                <button onClick={() => runAction("mute", post)}>Mute</button>
                <button onClick={() => runAction("unmute", post)}>Unmute</button>
                <button onClick={() => runAction("hide", post)}>Hide</button>
              </footer>
            </article>
          ))}
          <div ref={sentinelRef} className="scroll-sentinel">
            {loading ? "Loading..." : hasMore ? "Scroll for more" : "No more posts"}
          </div>
        </section>
      </main>
    </div>
  );
}

