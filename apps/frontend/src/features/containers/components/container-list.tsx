import { useMemo, useState } from "react";
import { ArrowUpDown, Box } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { useContainers } from "../hooks/use-containers";
import { ContainerCard } from "./container-card";
import { type ContainerFilter, ContainerFilters } from "./container-filters";

type SortKey = "name" | "state" | "image" | "ip" | "ports" | "resources";

export function ContainerList() {
  const [filter, setFilter] = useState<ContainerFilter>("all");
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");

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

  const sortedContainers = useMemo(() => {
    const direction = sortDirection === "asc" ? 1 : -1;
    const sortValue = (container: (typeof filteredContainers)[number]) => {
      switch (sortKey) {
        case "name":
          return container.name.toLowerCase();
        case "state":
          return container.state.toLowerCase();
        case "image":
          return container.image.toLowerCase();
        case "ip":
          return container.ipAddress ?? "";
        case "ports":
          return container.ports.length;
        case "resources":
          return container.networks.length + container.mounts.length;
        default:
          return container.name.toLowerCase();
      }
    };

    return [...filteredContainers].sort((a, b) => {
      const valueA = sortValue(a);
      const valueB = sortValue(b);
      if (typeof valueA === "number" && typeof valueB === "number") {
        return (valueA - valueB) * direction;
      }
      return String(valueA).localeCompare(String(valueB)) * direction;
    });
  }, [filteredContainers, sortDirection, sortKey]);

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(key);
    setSortDirection("asc");
  };

  const renderSortLabel = (label: string, key: SortKey) => (
    <button
      type="button"
      onClick={() => handleSort(key)}
      className="inline-flex items-center gap-1 hover:text-foreground"
    >
      {label}
      <ArrowUpDown className="h-3 w-3" />
    </button>
  );

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
            <table className="w-full min-w-[600px]">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground min-w-[140px]">
                    {renderSortLabel("Name", "name")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    {renderSortLabel("State", "state")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden md:table-cell">
                    Actions
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden lg:table-cell">
                    {renderSortLabel("Image", "image")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden xl:table-cell">
                    {renderSortLabel("IP", "ip")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden xl:table-cell">
                    {renderSortLabel("Ports", "ports")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden 2xl:table-cell">
                    {renderSortLabel("Resources", "resources")}
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden lg:table-cell">
                    Created
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground w-10"></th>
                </tr>
              </thead>
              <tbody>
                {sortedContainers.map((container) => (
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
