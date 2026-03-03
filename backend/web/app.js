const state = {
  timeline: "local",
  username: "alice",
  authenticated: false,
};

const elements = {
  healthDot: document.getElementById("healthDot"),
  healthLabel: document.getElementById("healthLabel"),

  authState: document.getElementById("authState"),
  authMsg: document.getElementById("authMsg"),
  loginForm: document.getElementById("loginForm"),
  loginUser: document.getElementById("loginUser"),
  loginPass: document.getElementById("loginPass"),
  registerForm: document.getElementById("registerForm"),
  regUsername: document.getElementById("regUsername"),
  regEmail: document.getElementById("regEmail"),
  regPassword: document.getElementById("regPassword"),
  logoutBtn: document.getElementById("logoutBtn"),

  profileName: document.getElementById("profileName"),
  profileHandle: document.getElementById("profileHandle"),
  profileActorUrl: document.getElementById("profileActorUrl"),
  statPosts: document.getElementById("statPosts"),
  statFollowers: document.getElementById("statFollowers"),
  statFollowing: document.getElementById("statFollowing"),

  composeForm: document.getElementById("composeForm"),
  postContent: document.getElementById("postContent"),
  visibility: document.getElementById("visibility"),
  username: document.getElementById("username"),
  postBtn: document.getElementById("postBtn"),
  composeMsg: document.getElementById("composeMsg"),

  followForm: document.getElementById("followForm"),
  followTarget: document.getElementById("followTarget"),
  unfollowBtn: document.getElementById("unfollowBtn"),
  followMsg: document.getElementById("followMsg"),

  localBtn: document.getElementById("localBtn"),
  homeBtn: document.getElementById("homeBtn"),
  refreshBtn: document.getElementById("refreshBtn"),
  timelineState: document.getElementById("timelineState"),
  timelineList: document.getElementById("timelineList"),

  groupCreateForm: document.getElementById("groupCreateForm"),
  groupSlug: document.getElementById("groupSlug"),
  groupTitle: document.getElementById("groupTitle"),
  groupJoinForm: document.getElementById("groupJoinForm"),
  groupJoinSlug: document.getElementById("groupJoinSlug"),
  groupLoadBtn: document.getElementById("groupLoadBtn"),
  groupTimelineList: document.getElementById("groupTimelineList"),
  groupMsg: document.getElementById("groupMsg"),

  checksGrid: document.getElementById("checksGrid"),
  notificationList: document.getElementById("notificationList"),
  markReadBtn: document.getElementById("markReadBtn"),
};

init();

function init() {
  elements.loginForm.addEventListener("submit", onLogin);
  elements.registerForm.addEventListener("submit", onRegister);
  elements.logoutBtn.addEventListener("click", onLogout);

  elements.username.addEventListener("input", onUsernameChange);
  elements.composeForm.addEventListener("submit", onSubmitPost);
  elements.postContent.addEventListener("keydown", onComposerKeydown);

  elements.followForm.addEventListener("submit", onFollow);
  elements.unfollowBtn.addEventListener("click", onUnfollow);

  elements.localBtn.addEventListener("click", () => switchTimeline("local"));
  elements.homeBtn.addEventListener("click", () => switchTimeline("home"));
  elements.refreshBtn.addEventListener("click", () => refreshAll());
  elements.timelineList.addEventListener("click", onTimelineAction);

  elements.groupCreateForm.addEventListener("submit", onCreateGroup);
  elements.groupJoinForm.addEventListener("submit", onJoinGroup);
  elements.groupLoadBtn.addEventListener("click", onLoadGroupTimeline);

  elements.markReadBtn.addEventListener("click", onMarkRead);

  refreshAll();
}

async function refreshAll() {
  await checkHealth();
  await loadAuthState();
  await Promise.all([loadChecks(), loadProfile(), loadTimeline(), loadNotifications()]);
}

async function checkHealth() {
  try {
    const res = await fetch("/healthz");
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    elements.healthDot.className = "dot ok";
    elements.healthLabel.textContent = "API status: healthy";
  } catch (err) {
    elements.healthDot.className = "dot fail";
    elements.healthLabel.textContent = `API status: failed (${err.message})`;
  }
}

async function loadAuthState() {
  try {
    const res = await fetch("/auth/me", { credentials: "same-origin" });
    if (!res.ok) throw new Error("not signed in");
    const payload = await res.json();
    state.authenticated = true;
    state.username = payload.actor.username;
    elements.username.value = state.username;
    elements.authState.textContent = `signed in as @${payload.actor.username}`;
    elements.authState.className = "mono";
    elements.loginForm.style.display = "none";
    elements.registerForm.style.display = "none";
  } catch {
    state.authenticated = false;
    elements.authState.textContent = "not signed in";
    elements.loginForm.style.display = "";
    elements.registerForm.style.display = "";
  }
}

async function onLogin(event) {
  event.preventDefault();
  const user = elements.loginUser.value.trim();
  const password = elements.loginPass.value;
  if (!user || !password) {
    setAuthMessage("Username/email and password are required.", true);
    return;
  }

  const body = user.includes("@")
    ? { email: user, password }
    : { username: user, password };

  try {
    const res = await fetch("/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      throw new Error(await safeErrorText(res));
    }
    elements.loginPass.value = "";
    setAuthMessage("Login successful.", false);
    await refreshAll();
  } catch (err) {
    setAuthMessage(`Login failed: ${err.message}`, true);
  }
}

async function onRegister(event) {
  event.preventDefault();
  const username = elements.regUsername.value.trim().toLowerCase();
  const email = elements.regEmail.value.trim();
  const password = elements.regPassword.value;
  if (!username || !email || !password) {
    setAuthMessage("Username, email, and password are required.", true);
    return;
  }

  try {
    const res = await fetch("/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ username, email, password }),
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    elements.regPassword.value = "";
    setAuthMessage("Account created.", false);
    await refreshAll();
  } catch (err) {
    setAuthMessage(`Register failed: ${err.message}`, true);
  }
}

async function onLogout() {
  try {
    await fetch("/auth/logout", { method: "POST", credentials: "same-origin" });
  } finally {
    setAuthMessage("Signed out.", false);
    await refreshAll();
  }
}

function setAuthMessage(text, isError) {
  elements.authMsg.textContent = text;
  elements.authMsg.className = `msg ${isError ? "error" : "ok"}`;
}

function onUsernameChange() {
  if (state.authenticated) {
    elements.username.value = state.username;
    return;
  }
  state.username = elements.username.value.trim() || "alice";
  loadProfile();
}

function onComposerKeydown(event) {
  if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
    event.preventDefault();
    elements.composeForm.requestSubmit();
  }
}

async function loadProfile() {
  const username = state.username || "alice";
  const actorPath = `/users/${encodeURIComponent(username)}`;
  const outboxPath = `/users/${encodeURIComponent(username)}/outbox`;
  const followersPath = `/users/${encodeURIComponent(username)}/followers`;
  const followingPath = `/users/${encodeURIComponent(username)}/following`;

  try {
    const [actorRes, outboxRes, followersRes, followingRes] = await Promise.all([
      fetch(actorPath),
      fetch(outboxPath),
      fetch(followersPath),
      fetch(followingPath),
    ]);
    if (!actorRes.ok) throw new Error(`actor HTTP ${actorRes.status}`);

    const actor = await actorRes.json();
    const outbox = outboxRes.ok ? await outboxRes.json() : { totalItems: 0 };
    const followers = followersRes.ok ? await followersRes.json() : { totalItems: 0 };
    const following = followingRes.ok ? await followingRes.json() : { totalItems: 0 };

    elements.profileName.textContent = actor.name || username;
    elements.profileHandle.textContent = `@${actor.preferredUsername || username}`;
    elements.profileActorUrl.textContent = actor.id || actorPath;
    elements.statPosts.textContent = String(outbox.totalItems ?? 0);
    elements.statFollowers.textContent = String(followers.totalItems ?? 0);
    elements.statFollowing.textContent = String(following.totalItems ?? 0);
  } catch {
    elements.profileName.textContent = username;
    elements.profileHandle.textContent = `@${username}`;
    elements.profileActorUrl.textContent = `${window.location.origin}${actorPath}`;
    elements.statPosts.textContent = "0";
    elements.statFollowers.textContent = "0";
    elements.statFollowing.textContent = "0";
  }
}

async function onSubmitPost(event) {
  event.preventDefault();
  const content = elements.postContent.value.trim();
  const visibility = elements.visibility.value;
  if (!content) {
    setComposeMessage("Post content is required.", true);
    return;
  }

  elements.postBtn.disabled = true;
  setComposeMessage("Publishing...", false);
  try {
    const res = await fetch("/api/v1/posts", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ content, visibility }),
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    elements.postContent.value = "";
    setComposeMessage("Published.", false);
    await Promise.all([loadProfile(), loadTimeline(), loadChecks(), loadNotifications()]);
  } catch (err) {
    setComposeMessage(`Publish failed: ${err.message}`, true);
  } finally {
    elements.postBtn.disabled = false;
  }
}

function setComposeMessage(text, isError) {
  elements.composeMsg.textContent = text;
  elements.composeMsg.className = `msg ${isError ? "error" : "ok"}`;
}

async function onFollow(event) {
  event.preventDefault();
  const target = elements.followTarget.value.trim();
  if (!target) {
    setFollowMessage("Target required.", true);
    return;
  }
  try {
    const res = await fetch("/api/v1/follows", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ target }),
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    setFollowMessage("Follow request sent.", false);
    await Promise.all([loadProfile(), loadNotifications()]);
  } catch (err) {
    setFollowMessage(`Follow failed: ${err.message}`, true);
  }
}

async function onUnfollow() {
  const target = elements.followTarget.value.trim();
  if (!target) {
    setFollowMessage("Target required.", true);
    return;
  }
  try {
    const res = await fetch("/api/v1/unfollow", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ target }),
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    setFollowMessage("Unfollowed.", false);
    await loadProfile();
  } catch (err) {
    setFollowMessage(`Unfollow failed: ${err.message}`, true);
  }
}

function setFollowMessage(text, isError) {
  elements.followMsg.textContent = text;
  elements.followMsg.className = `msg ${isError ? "error" : "ok"}`;
}

function switchTimeline(type) {
  state.timeline = type;
  elements.localBtn.classList.toggle("active", type === "local");
  elements.localBtn.setAttribute("aria-selected", String(type === "local"));
  elements.homeBtn.classList.toggle("active", type === "home");
  elements.homeBtn.setAttribute("aria-selected", String(type === "home"));
  loadTimeline();
}

async function loadTimeline() {
  const username = encodeURIComponent(state.username || "alice");
  const path = state.timeline === "home"
    ? `/api/v1/timelines/home?username=${username}&limit=25`
    : "/api/v1/timelines/local?limit=25";

  setTimelineState("loading...");
  try {
    const res = await fetch(path, { credentials: "same-origin" });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const payload = await res.json();
    renderTimeline(Array.isArray(payload.items) ? payload.items : []);
    setTimelineState(`${state.timeline} timeline`);
  } catch (err) {
    elements.timelineList.innerHTML = "";
    setTimelineState(`error: ${err.message}`);
  }
}

function renderTimeline(items) {
  elements.timelineList.innerHTML = "";
  if (items.length === 0) {
    const li = document.createElement("li");
    li.className = "timeline-item";
    li.textContent = "No notices yet.";
    elements.timelineList.appendChild(li);
    return;
  }

  for (const item of items) {
    const li = document.createElement("li");
    li.className = "timeline-item";
    const username = item.username || "unknown";
    const published = item.published_at ? formatDate(item.published_at) : "-";
    const body = item.content_text || item.content_html || "";
    const noteID = Number(item.id);
    li.innerHTML = `
      <div class="meta">
        <span>@${escapeHTML(username)}</span>
        <time>${escapeHTML(published)}</time>
      </div>
      <div class="content">${escapeHTML(body)}</div>
      <div class="row">
        <button type="button" data-action="like" data-id="${noteID}">Like</button>
        <button type="button" data-action="boost" data-id="${noteID}">Boost</button>
        <button type="button" class="ghost" data-action="delete" data-id="${noteID}">Delete</button>
      </div>
    `;
    elements.timelineList.appendChild(li);
  }
}

async function onTimelineAction(event) {
  const button = event.target.closest("button[data-action]");
  if (!button) return;
  const noteID = Number(button.dataset.id);
  if (!Number.isFinite(noteID) || noteID <= 0) return;
  const action = button.dataset.action;

  try {
    if (action === "like") await postAction(`/api/v1/notes/${noteID}/like`, "POST");
    if (action === "boost") await postAction(`/api/v1/notes/${noteID}/boost`, "POST");
    if (action === "delete") await postAction(`/api/v1/posts/${noteID}`, "DELETE");
    await Promise.all([loadTimeline(), loadNotifications()]);
  } catch (err) {
    setTimelineState(`action failed: ${err.message}`);
  }
}

async function postAction(path, method) {
  const res = await fetch(path, { method, credentials: "same-origin" });
  if (!res.ok) throw new Error(await safeErrorText(res));
}

async function onCreateGroup(event) {
  event.preventDefault();
  const slug = elements.groupSlug.value.trim().toLowerCase();
  const title = elements.groupTitle.value.trim();
  if (!slug || !title) {
    setGroupMessage("Slug and title are required.", true);
    return;
  }
  try {
    const res = await fetch("/api/v1/groups", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ slug, title }),
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    elements.groupJoinSlug.value = slug;
    setGroupMessage("Group created.", false);
  } catch (err) {
    setGroupMessage(`Create failed: ${err.message}`, true);
  }
}

async function onJoinGroup(event) {
  event.preventDefault();
  const slug = elements.groupJoinSlug.value.trim().toLowerCase();
  if (!slug) {
    setGroupMessage("Join slug is required.", true);
    return;
  }
  try {
    const res = await fetch(`/api/v1/groups/${encodeURIComponent(slug)}/join`, {
      method: "POST",
      credentials: "same-origin",
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    setGroupMessage("Joined group.", false);
  } catch (err) {
    setGroupMessage(`Join failed: ${err.message}`, true);
  }
}

async function onLoadGroupTimeline() {
  const slug = elements.groupJoinSlug.value.trim().toLowerCase();
  if (!slug) {
    setGroupMessage("Timeline slug is required.", true);
    return;
  }
  try {
    const res = await fetch(`/api/v1/groups/${encodeURIComponent(slug)}/timeline?limit=25`, {
      credentials: "same-origin",
    });
    if (!res.ok) throw new Error(await safeErrorText(res));
    const payload = await res.json();
    renderGroupTimeline(Array.isArray(payload.items) ? payload.items : []);
    setGroupMessage(`Loaded ${slug}.`, false);
  } catch (err) {
    setGroupMessage(`Load failed: ${err.message}`, true);
  }
}

function renderGroupTimeline(items) {
  elements.groupTimelineList.innerHTML = "";
  if (items.length === 0) {
    const li = document.createElement("li");
    li.className = "timeline-item";
    li.textContent = "No group posts yet.";
    elements.groupTimelineList.appendChild(li);
    return;
  }
  for (const item of items) {
    const li = document.createElement("li");
    li.className = "timeline-item";
    li.innerHTML = `
      <div class="meta">
        <span>@${escapeHTML(item.username || "unknown")}</span>
        <time>${escapeHTML(formatDate(item.published_at || ""))}</time>
      </div>
      <div class="content">${escapeHTML(item.content_text || item.content_html || "")}</div>
    `;
    elements.groupTimelineList.appendChild(li);
  }
}

function setGroupMessage(text, isError) {
  elements.groupMsg.textContent = text;
  elements.groupMsg.className = `msg ${isError ? "error" : "ok"}`;
}

async function loadNotifications() {
  if (!state.authenticated) {
    elements.notificationList.innerHTML = "<li class=\"timeline-item\">Sign in to view notifications.</li>";
    return;
  }
  try {
    const res = await fetch("/api/v1/notifications?limit=25", { credentials: "same-origin" });
    if (!res.ok) throw new Error(await safeErrorText(res));
    const payload = await res.json();
    const items = Array.isArray(payload.items) ? payload.items : [];
    renderNotifications(items);
  } catch (err) {
    elements.notificationList.innerHTML = `<li class="timeline-item">Failed: ${escapeHTML(err.message)}</li>`;
  }
}

function renderNotifications(items) {
  elements.notificationList.innerHTML = "";
  if (items.length === 0) {
    elements.notificationList.innerHTML = "<li class=\"timeline-item\">No notifications.</li>";
    return;
  }
  for (const item of items) {
    const li = document.createElement("li");
    li.className = "timeline-item";
    const actor = item.username ? `@${item.username}` : "someone";
    li.innerHTML = `
      <div class="meta">
        <span>${escapeHTML(item.kind)}</span>
        <time>${escapeHTML(formatDate(item.created_at))}</time>
      </div>
      <div class="content">${escapeHTML(actor)} ${escapeHTML(notificationText(item.kind))}</div>
    `;
    elements.notificationList.appendChild(li);
  }
}

function notificationText(kind) {
  if (kind === "follow") return "followed you";
  if (kind === "like") return "liked your post";
  if (kind === "announce") return "boosted your post";
  if (kind === "reply") return "replied to your post";
  return "interacted";
}

async function onMarkRead() {
  if (!state.authenticated) return;
  await fetch("/api/v1/notifications/read-all", {
    method: "POST",
    credentials: "same-origin",
  });
  await loadNotifications();
}

async function loadChecks() {
  elements.checksGrid.innerHTML = "";
  const checks = [
    { name: "Health", url: "/healthz" },
    { name: "NodeInfo", url: "/.well-known/nodeinfo" },
    { name: "Actor", url: `/users/${encodeURIComponent(state.username || "alice")}` },
    { name: "Outbox", url: `/users/${encodeURIComponent(state.username || "alice")}/outbox` },
  ];

  const cards = checks.map((check) => {
    const card = document.createElement("article");
    card.className = "check-card";
    card.innerHTML = `
      <div class="check-head">
        <span>${escapeHTML(check.name)}</span>
        <span class="status-pill status-pending">pending</span>
      </div>
      <p class="check-url">${escapeHTML(check.url)}</p>
    `;
    elements.checksGrid.appendChild(card);
    return { ...check, card };
  });

  await Promise.all(cards.map(runCheck));
}

async function runCheck(check) {
  const pill = check.card.querySelector(".status-pill");
  try {
    const res = await fetch(check.url);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    pill.textContent = `ok ${res.status}`;
    pill.className = "status-pill status-ok";
  } catch {
    pill.textContent = "fail";
    pill.className = "status-pill status-fail";
  }
}

function setTimelineState(text) {
  elements.timelineState.textContent = text;
}

function formatDate(input) {
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) return input;
  return date.toLocaleString();
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

async function safeErrorText(res) {
  try {
    const body = await res.json();
    if (body && body.error) return body.error;
  } catch (_) {
  }
  return `HTTP ${res.status}`;
}
