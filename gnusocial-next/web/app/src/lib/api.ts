import { fetchNotifications, fetchThread, fetchTimeline } from "./mockData";
import type { NotificationItem, ThreadPayload, TimelineKind } from "./types";

export async function getTimeline(kind: TimelineKind, cursor: string | null) {
  return fetchTimeline(kind, cursor, 20);
}

export async function getThread(threadId: string): Promise<ThreadPayload> {
  return fetchThread(threadId);
}

export async function getNotifications(): Promise<NotificationItem[]> {
  return fetchNotifications();
}
