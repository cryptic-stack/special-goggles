import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { AdminPage } from "../pages/AdminPage";
import { BookmarksPage } from "../pages/BookmarksPage";
import { ExplorePage } from "../pages/ExplorePage";
import { FederatedPage } from "../pages/FederatedPage";
import { FollowingPage } from "../pages/FollowingPage";
import { HomePage } from "../pages/HomePage";
import { ListsPage } from "../pages/ListsPage";
import { LocalPage } from "../pages/LocalPage";
import { MessagesPage } from "../pages/MessagesPage";
import { ModerationPage } from "../pages/ModerationPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { NotificationsPage } from "../pages/NotificationsPage";
import { ProfilePage } from "../pages/ProfilePage";
import { SearchPage } from "../pages/SearchPage";
import { SettingsPage } from "../pages/SettingsPage";
import { ThreadPage } from "../pages/ThreadPage";
import { UiKitPage } from "../pages/UiKitPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/home" replace /> },
      { path: "home", element: <HomePage /> },
      { path: "following", element: <FollowingPage /> },
      { path: "local", element: <LocalPage /> },
      { path: "federated", element: <FederatedPage /> },
      { path: "explore", element: <ExplorePage /> },
      { path: "search", element: <SearchPage /> },
      { path: "notifications", element: <NotificationsPage /> },
      { path: "messages", element: <MessagesPage /> },
      { path: "lists", element: <ListsPage /> },
      { path: "bookmarks", element: <BookmarksPage /> },
      { path: "profile", element: <ProfilePage /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "moderation", element: <ModerationPage /> },
      { path: "admin", element: <AdminPage /> },
      { path: "thread/:threadId", element: <ThreadPage /> },
      { path: "ui-kit", element: <UiKitPage /> },
      { path: "*", element: <NotFoundPage /> }
    ]
  }
]);
