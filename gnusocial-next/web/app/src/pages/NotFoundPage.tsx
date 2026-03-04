import { Link } from "react-router-dom";
import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";

export function NotFoundPage() {
  return (
    <>
      <TopBar title="Not found" />
      <div className="mx-auto max-w-feed p-4">
        <EmptyState
          title="Route not found"
          description="The route does not exist in this milestone."
          action={
            <Link to="/home" className="rounded-md border border-ui-line px-3 py-2 text-sm">
              Go Home
            </Link>
          }
        />
      </div>
    </>
  );
}
