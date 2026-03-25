import { useState } from "react";
import { Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import {
  type AuditFilter,
  type WebhookPayloadsFilter,
  useAuditLogs,
  useWebhookPayloads,
} from "../hooks/use-audit";
import type { AuditTab } from "./audit-constants";
import { AuditFiltersBar } from "./audit-filters-bar";
import { AuditListSkeleton } from "./audit-list-skeleton";
import { AuditPagination } from "./audit-pagination";
import { PlatformEventsTable } from "./platform-events-table";
import { WebhookEventsTable } from "./webhook-events-table";

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
    return <AuditListSkeleton />;
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

      <AuditFiltersBar
        activeTab={activeTab}
        search={search}
        onSearchChange={setSearch}
        eventType={filter.eventType}
        onEventTypeChange={handleEventTypeChange}
        resourceType={filter.resourceType}
        onResourceTypeChange={handleResourceTypeChange}
      />

      {activeTab === "webhooks" && webhookData?.payloads?.length === 0 && (
        <EmptyState
          icon={Activity}
          title="No webhook events found"
          description="Webhook payloads will appear here when GitHub sends events. Ensure the webhook URL is configured and publicly reachable."
        />
      )}
      {activeTab === "webhooks" && (webhookData?.payloads?.length ?? 0) > 0 && (
        <WebhookEventsTable payloads={webhookData?.payloads ?? []} />
      )}

      {activeTab === "platform" && filteredLogs?.length === 0 && (
        <EmptyState
          icon={Activity}
          title="No audit logs found"
          description="No events match your current filters."
        />
      )}
      {activeTab === "platform" && (filteredLogs?.length ?? 0) > 0 && (
        <PlatformEventsTable logs={filteredLogs ?? []} />
      )}

      {displayTotal > 0 && (
        <AuditPagination
          total={displayTotal}
          offset={displayOffset}
          limit={displayLimit}
          currentPage={displayCurrentPage}
          totalPages={displayTotalPages}
          label={activeTab === "platform" ? "events" : "webhooks"}
          onPrevPage={handlePrevPage}
          onNextPage={handleNextPage}
        />
      )}
    </div>
  );
}
