import { useState } from "react";
import {
  Activity,
  ChevronLeft,
  ChevronRight,
  Filter,
  Search,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import {
  type AuditFilter,
  type WebhookPayloadsFilter,
  useAuditLogs,
  useWebhookPayloads,
} from "../hooks/use-audit";

type AuditTab = "platform" | "webhooks";

const EVENT_TYPES = [
  { value: "app.created", label: "App Created" },
  { value: "app.deleted", label: "App Deleted" },
  { value: "app.purged", label: "App Purged" },
  { value: "deploy.started", label: "Deploy Started" },
  { value: "deploy.success", label: "Deploy Success" },
  { value: "deploy.failed", label: "Deploy Failed" },
  { value: "env.created", label: "Env Created" },
  { value: "env.updated", label: "Env Updated" },
  { value: "env.deleted", label: "Env Deleted" },
  { value: "domain.added", label: "Domain Added" },
  { value: "domain.removed", label: "Domain Removed" },
  { value: "container.started", label: "Container Started" },
  { value: "container.stopped", label: "Container Stopped" },
  { value: "container.removed", label: "Container Removed" },
  { value: "user.logged_in", label: "User Login" },
  { value: "user.logged_out", label: "User Logout" },
  { value: "image.removed", label: "Image Removed" },
  { value: "images.pruned", label: "Images Pruned" },
];

const RESOURCE_TYPES = [
  { value: "app", label: "App" },
  { value: "deployment", label: "Deployment" },
  { value: "env_var", label: "Environment" },
  { value: "domain", label: "Domain" },
  { value: "container", label: "Container" },
  { value: "user", label: "User" },
  { value: "image", label: "Image" },
];

function getEventBadgeColor(eventType: string): string {
  if (
    eventType.includes("webhook.deployment_queued") ||
    eventType.includes("deployment_queued") ||
    eventType.includes("pong")
  ) {
    return "bg-green-500/20 text-green-400 border-green-500/30";
  }
  if (
    eventType.includes("webhook.invalid") ||
    eventType.includes("invalid_signature") ||
    eventType.includes("parse_error") ||
    eventType.includes("webhook.error")
  ) {
    return "bg-red-500/20 text-red-400 border-red-500/30";
  }
  if (
    eventType.includes("created") ||
    eventType.includes("added") ||
    eventType.includes("received")
  ) {
    return "bg-green-500/20 text-green-400 border-green-500/30";
  }
  if (
    eventType.includes("deleted") ||
    eventType.includes("removed") ||
    eventType.includes("purged")
  ) {
    return "bg-red-500/20 text-red-400 border-red-500/30";
  }
  if (eventType.includes("success") || eventType.includes("logged_in")) {
    return "bg-blue-500/20 text-blue-400 border-blue-500/30";
  }
  if (eventType.includes("failed") || eventType.includes("error")) {
    return "bg-orange-500/20 text-orange-400 border-orange-500/30";
  }
  if (eventType.includes("ignored")) {
    return "bg-gray-500/20 text-gray-400 border-gray-500/30";
  }
  return "bg-gray-500/20 text-gray-400 border-gray-500/30";
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function AuditList() {
  const [activeTab, setActiveTab] = useState<AuditTab>("platform");
  const [filter, setFilter] = useState<AuditFilter>({
    limit: 25,
    offset: 0,
  });
  const [webhookFilter, setWebhookFilter] = useState<WebhookPayloadsFilter>({
    limit: 25,
    offset: 0,
  });
  const [search, setSearch] = useState("");

  const { data, isLoading, error } = useAuditLogs(filter, {
    enabled: activeTab === "platform",
  });
  const {
    data: webhookData,
    isLoading: webhookLoading,
    error: webhookError,
  } = useWebhookPayloads(webhookFilter, { enabled: activeTab === "webhooks" });

  const handleEventTypeChange = (value: string) => {
    setFilter((prev) => ({
      ...prev,
      eventType: value === "all" ? undefined : value,
      offset: 0,
    }));
  };

  const handleResourceTypeChange = (value: string) => {
    setFilter((prev) => ({
      ...prev,
      resourceType: value === "all" ? undefined : value,
      offset: 0,
    }));
  };

  const handlePrevPage = () => {
    if (activeTab === "webhooks") {
      setWebhookFilter((prev) => ({
        ...prev,
        offset: Math.max(0, (prev.offset ?? 0) - (prev.limit ?? 25)),
      }));
    } else {
      setFilter((prev) => ({
        ...prev,
        offset: Math.max(0, (prev.offset ?? 0) - (prev.limit ?? 25)),
      }));
    }
  };

  const handleNextPage = () => {
    if (activeTab === "webhooks") {
      setWebhookFilter((prev) => ({
        ...prev,
        offset: (prev.offset ?? 0) + (prev.limit ?? 25),
      }));
    } else {
      setFilter((prev) => ({
        ...prev,
        offset: (prev.offset ?? 0) + (prev.limit ?? 25),
      }));
    }
  };

  const isLoadingData = activeTab === "platform" ? isLoading : webhookLoading;
  const hasError = activeTab === "platform" ? error : webhookError;

  if (isLoadingData) {
    return (
      <div className="space-y-4">
        <div className="flex gap-4">
          <Skeleton className="h-10 w-[200px]" />
          <Skeleton className="h-10 w-[200px]" />
        </div>
        <Card>
          <div className="p-4 space-y-3">
            {Array.from({ length: 10 }).map((_, i) => (
              <Skeleton
                key={`audit-skeleton-${i.toString()}`}
                className="h-12 w-full"
              />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (hasError) {
    return (
      <ErrorMessage
        message={
          activeTab === "platform"
            ? "Failed to load audit logs"
            : "Failed to load webhook payloads"
        }
      />
    );
  }

  const filteredLogs = data?.logs.filter((log) => {
    if (!search) return true;
    const searchLower = search.toLowerCase();
    return (
      log.eventType.toLowerCase().includes(searchLower) ||
      log.resourceType.toLowerCase().includes(searchLower) ||
      log.resourceName?.toLowerCase().includes(searchLower) ||
      log.userName?.toLowerCase().includes(searchLower)
    );
  });

  const currentPage =
    Math.floor((filter.offset ?? 0) / (filter.limit ?? 25)) + 1;
  const totalPages = Math.ceil((data?.total ?? 0) / (filter.limit ?? 25));
  const webhookCurrentPage =
    Math.floor((webhookFilter.offset ?? 0) / (webhookFilter.limit ?? 25)) + 1;
  const webhookTotalPages = Math.ceil(
    (webhookData?.total ?? 0) / (webhookFilter.limit ?? 25),
  );
  const displayTotal =
    activeTab === "platform" ? (data?.total ?? 0) : (webhookData?.total ?? 0);
  const displayOffset =
    activeTab === "platform"
      ? (filter.offset ?? 0)
      : (webhookFilter.offset ?? 0);
  const displayLimit =
    activeTab === "platform"
      ? (filter.limit ?? 25)
      : (webhookFilter.limit ?? 25);
  const displayCurrentPage =
    activeTab === "platform" ? currentPage : webhookCurrentPage;
  const displayTotalPages =
    activeTab === "platform" ? totalPages : webhookTotalPages;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2 border-b border-border pb-2">
        <Button
          variant={activeTab === "platform" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("platform")}
        >
          Platform Events
        </Button>
        <Button
          variant={activeTab === "webhooks" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("webhooks")}
        >
          Webhook Events
        </Button>
      </div>
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search logs..."
            className="pl-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-2">
          {activeTab === "platform" && (
            <Select
              value={filter.eventType ?? "all"}
              onValueChange={handleEventTypeChange}
            >
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
              value={filter.resourceType ?? "all"}
              onValueChange={handleResourceTypeChange}
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

      {activeTab === "webhooks" && webhookData?.payloads?.length === 0 && (
        <EmptyState
          icon={Activity}
          title="No webhook events found"
          description="Webhook payloads will appear here when GitHub sends events. Ensure the webhook URL is configured and publicly reachable."
        />
      )}
      {activeTab === "webhooks" && (webhookData?.payloads?.length ?? 0) > 0 && (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Event
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Delivery ID
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Outcome
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden md:table-cell">
                    Error
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody>
                {webhookData?.payloads?.map((p) => (
                  <tr
                    key={p.id}
                    className="border-b border-border hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4">
                      <Badge
                        variant="outline"
                        className={`text-xs ${getEventBadgeColor("webhook." + p.outcome)}`}
                      >
                        {p.eventType}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 font-mono text-xs">
                      {p.deliveryId}
                    </td>
                    <td className="py-3 px-4">
                      <Badge
                        variant="outline"
                        className={`text-xs ${getEventBadgeColor("webhook." + p.outcome)}`}
                      >
                        {p.outcome}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 hidden md:table-cell text-sm text-muted-foreground max-w-[200px] truncate">
                      {p.errorMessage ?? "-"}
                    </td>
                    <td className="py-3 px-4">
                      <span className="text-sm text-muted-foreground">
                        {formatDate(p.createdAt)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
      {activeTab === "platform" && filteredLogs?.length === 0 && (
        <EmptyState
          icon={Activity}
          title="No audit logs found"
          description="No events match your current filters."
        />
      )}
      {activeTab === "platform" && (filteredLogs?.length ?? 0) > 0 && (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Event
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Resource
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden md:table-cell">
                    User
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground hidden lg:table-cell">
                    IP
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredLogs?.map((log) => (
                  <tr
                    key={log.id}
                    className="border-b border-border hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4">
                      <Badge
                        variant="outline"
                        className={`text-xs ${getEventBadgeColor(log.eventType)}`}
                      >
                        {log.eventType.replace(".", " ").replace("_", " ")}
                      </Badge>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex flex-col">
                        <span className="text-sm font-medium">
                          {log.resourceName ?? log.resourceId ?? "-"}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {log.resourceType}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-4 hidden md:table-cell">
                      <span className="text-sm text-muted-foreground">
                        {log.userName ?? "-"}
                      </span>
                    </td>
                    <td className="py-3 px-4 hidden lg:table-cell">
                      <span className="text-sm text-muted-foreground font-mono">
                        {log.ipAddress ?? "-"}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <span className="text-sm text-muted-foreground">
                        {formatDate(log.createdAt)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">
          Showing {displayOffset + 1} -{" "}
          {Math.min(displayOffset + displayLimit, displayTotal)} of{" "}
          {displayTotal} {activeTab === "platform" ? "events" : "webhooks"}
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handlePrevPage}
            disabled={displayOffset === 0}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <span className="text-sm">
            Page {displayCurrentPage} of {displayTotalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={handleNextPage}
            disabled={displayOffset + displayLimit >= displayTotal}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
