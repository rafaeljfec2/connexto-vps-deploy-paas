import { Server as ServerIcon } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { useServers } from "../hooks/use-servers";
import { ServerCard } from "./server-card";

export function ServerList() {
  const { data: servers, isLoading, error } = useServers();

  if (isLoading) {
    return (
      <div
        role="status"
        aria-live="polite"
        aria-busy="true"
        className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"
      >
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-40 w-full rounded-xl" />
        ))}
        <span className="sr-only">Loading servers…</span>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage message="Failed to load servers" />;
  }

  if (!servers?.length) {
    return (
      <EmptyState
        icon={ServerIcon}
        title="No servers"
        description="Add a remote server to enable deploy to other machines."
      />
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {servers.map((server) => (
        <ServerCard key={server.id} server={server} />
      ))}
    </div>
  );
}
