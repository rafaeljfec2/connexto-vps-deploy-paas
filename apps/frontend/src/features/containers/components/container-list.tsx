import { useMemo, useState } from "react";
import { Box } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { useContainers } from "../hooks/use-containers";
import { ContainerCard } from "./container-card";
import { type ContainerFilter, ContainerFilters } from "./container-filters";

export function ContainerList() {
  const [filter, setFilter] = useState<ContainerFilter>("all");
  const [search, setSearch] = useState("");

  const { data: containers, isLoading, error } = useContainers(true);

  const filteredContainers = useMemo(() => {
    if (!containers) return [];

    return containers.filter((container) => {
      const matchesSearch =
        search === "" ||
        container.name.toLowerCase().includes(search.toLowerCase()) ||
        container.image.toLowerCase().includes(search.toLowerCase());

      const matchesFilter = (() => {
        switch (filter) {
          case "running":
            return container.state === "running";
          case "stopped":
            return container.state === "exited" || container.state === "dead";
          case "flowdeploy":
            return container.isFlowDeployManaged;
          default:
            return true;
        }
      })();

      return matchesSearch && matchesFilter;
    });
  }, [containers, filter, search]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex gap-4">
          <Skeleton className="h-10 w-[300px]" />
          <Skeleton className="h-10 w-[250px]" />
        </div>
        <Card>
          <div className="p-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton
                key={`container-skeleton-${i.toString()}`}
                className="h-16 w-full"
              />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage message="Failed to load containers" />;
  }

  return (
    <div className="space-y-4">
      <ContainerFilters
        filter={filter}
        onFilterChange={setFilter}
        search={search}
        onSearchChange={setSearch}
      />

      {filteredContainers.length === 0 ? (
        <EmptyState
          icon={Box}
          title="No containers found"
          description={
            search || filter !== "all"
              ? "Try adjusting your filters or search query."
              : "No Docker containers are running on this server."
          }
        />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Name
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    State
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden md:table-cell">
                    Actions
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden lg:table-cell">
                    Image
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden xl:table-cell">
                    IP
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden xl:table-cell">
                    Ports
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden 2xl:table-cell">
                    Resources
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground w-10"></th>
                </tr>
              </thead>
              <tbody>
                {filteredContainers.map((container) => (
                  <ContainerCard key={container.id} container={container} />
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <div className="text-sm text-muted-foreground">
        Showing {filteredContainers.length} of {containers?.length ?? 0}{" "}
        containers
      </div>
    </div>
  );
}
