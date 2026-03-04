import type { Post, User } from "./types";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

type RequestOptions = {
  method?: string;
  body?: unknown;
};

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    method: options.method ?? "GET",
    headers: {
      "Content-Type": "application/json"
    },
    body: options.body ? JSON.stringify(options.body) : undefined
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function fetchPublicTimeline(offset = 0, limit = 20, viewerId?: string): Promise<Post[]> {
  const params = new URLSearchParams({
    offset: String(offset),
    limit: String(limit)
  });
  if (viewerId) {
    params.set("viewerId", viewerId);
  }
  return request<Post[]>(`/v1/timeline/public?${params.toString()}`);
}

export async function fetchHomeTimeline(userId: string, offset = 0, limit = 20): Promise<Post[]> {
  const params = new URLSearchParams({
    userId,
    offset: String(offset),
    limit: String(limit)
  });
  return request<Post[]>(`/v1/timeline/home?${params.toString()}`);
}

export async function createUser(payload: {
  username: string;
  email: string;
  password: string;
  displayName: string;
  bio?: string;
  avatarUrl?: string;
}): Promise<User> {
  return request<User>("/v1/users", { method: "POST", body: payload });
}

export async function lookupUser(username: string): Promise<User> {
  return request<User>(`/v1/users/${encodeURIComponent(username)}`);
}

export async function createPost(payload: {
  authorId: string;
  content: string;
  visibility: string;
}): Promise<Post> {
  return request<Post>("/v1/status", { method: "POST", body: payload });
}

export async function follow(followerId: string, followedId: string): Promise<void> {
  await request("/v1/users/follow", { method: "POST", body: { followerId, followedId } });
}

export async function unfollow(followerId: string, followedId: string): Promise<void> {
  await request(`/v1/users/follow?followerId=${followerId}&followedId=${followedId}`, { method: "DELETE" });
}

export async function mute(muterId: string, mutedId: string): Promise<void> {
  await request("/v1/users/mute", { method: "POST", body: { followerId: muterId, followedId: mutedId } });
}

export async function unmute(muterId: string, mutedId: string): Promise<void> {
  await request(`/v1/users/mute?muterId=${muterId}&mutedId=${mutedId}`, { method: "DELETE" });
}

export async function hidePost(userId: string, postId: string): Promise<void> {
  await request(`/v1/users/hide-post?userId=${userId}&postId=${postId}`, { method: "POST" });
}

