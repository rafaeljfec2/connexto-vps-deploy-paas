import { Filter, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { AuditTab } from "./audit-constants";
import { EVENT_TYPES, RESOURCE_TYPES } from "./audit-constants";

interface AuditFiltersBarProps {
  readonly activeTab: AuditTab;
  readonly search: string;
  readonly onSearchChange: (value: string) => void;
  readonly eventType?: string;
  readonly onEventTypeChange: (value: string) => void;
  readonly resourceType?: string;
  readonly onResourceTypeChange: (value: string) => void;
}

export function AuditFiltersBar({
  activeTab,
  search,
  onSearchChange,
  eventType,
  onEventTypeChange,
  resourceType,
  onResourceTypeChange,
}: Readonly<AuditFiltersBarProps>) {
  return (
    <div className="flex flex-col sm:flex-row gap-4">
      <div className="relative flex-1">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search logs..."
          className="pl-9"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <div className="flex gap-2">
        {activeTab === "platform" && (
          <Select value={eventType ?? "all"} onValueChange={onEventTypeChange}>
            <SelectTrigger className="w-[180px]">
              <Filter className="h-4 w-4 mr-2" />
              <SelectValue placeholder="Event Type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Events</SelectItem>
              {EVENT_TYPES.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  {type.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        {activeTab === "platform" && (
          <Select
            value={resourceType ?? "all"}
            onValueChange={onResourceTypeChange}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="Resource" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Resources</SelectItem>
              {RESOURCE_TYPES.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  {type.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>
    </div>
  );
}
