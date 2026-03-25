import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { formatDateWithSeconds } from "@/lib/format";
import type { WebhookPayload } from "../hooks/use-audit";
import { getEventBadgeColor } from "./audit-utils";

interface WebhookEventsTableProps {
  readonly payloads: readonly WebhookPayload[];
}

export function WebhookEventsTable({
  payloads,
}: Readonly<WebhookEventsTableProps>) {
  return (
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
            {payloads.map((p) => (
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
                <td className="py-3 px-4 font-mono text-xs">{p.deliveryId}</td>
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
                    {formatDateWithSeconds(p.createdAt)}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
