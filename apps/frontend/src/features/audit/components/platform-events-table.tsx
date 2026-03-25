import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { formatDateWithSeconds } from "@/lib/format";
import type { AuditLog } from "../hooks/use-audit";
import { getEventBadgeColor } from "./audit-utils";

interface PlatformEventsTableProps {
  readonly logs: readonly AuditLog[];
}

export function PlatformEventsTable({
  logs,
}: Readonly<PlatformEventsTableProps>) {
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
            {logs.map((log) => (
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
                    {formatDateWithSeconds(log.createdAt)}
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
