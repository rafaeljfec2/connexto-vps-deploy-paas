import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

export type ContainerFilter = "all" | "running" | "stopped" | "flowdeploy";

interface ContainerFiltersProps {
  readonly filter: ContainerFilter;
  readonly onFilterChange: (filter: ContainerFilter) => void;
  readonly search: string;
  readonly onSearchChange: (search: string) => void;
}

export function ContainerFilters({
  filter,
  onFilterChange,
  search,
  onSearchChange,
}: ContainerFiltersProps) {
  return (
    <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
      <Tabs
        value={filter}
        onValueChange={(v) => onFilterChange(v as ContainerFilter)}
      >
        <TabsList>
          <TabsTrigger value="all" className="text-xs sm:text-sm">
            All
          </TabsTrigger>
          <TabsTrigger value="running" className="text-xs sm:text-sm">
            Running
          </TabsTrigger>
          <TabsTrigger value="stopped" className="text-xs sm:text-sm">
            Stopped
          </TabsTrigger>
          <TabsTrigger value="flowdeploy" className="text-xs sm:text-sm">
            FlowDeploy
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <div className="relative w-full sm:w-auto">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search containers..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9 w-full sm:w-[250px]"
        />
      </div>
    </div>
  );
}
