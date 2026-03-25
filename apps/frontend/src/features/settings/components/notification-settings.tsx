import { useQuery } from "@tanstack/react-query";
import { Bell, Loader2 } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { api } from "@/services/api";
import { ChannelCard, CreateChannelDialog } from "./notifications";

export function NotificationSettings() {
  const { data: channels = [], isLoading } = useQuery({
    queryKey: ["notification-channels"],
    queryFn: () => api.notifications.channels.list(),
  });

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5" />
              Notifications
            </CardTitle>
            <CardDescription>
              Configure Slack, Discord, or Email channels to receive deploy and
              health alerts
            </CardDescription>
          </div>
          <CreateChannelDialog onSuccess={() => {}} />
        </div>
      </CardHeader>
      <CardContent>
        {channels.length === 0 ? (
          <div className="rounded-lg border border-dashed py-12 text-center">
            <Bell className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <p className="mt-2 text-sm font-medium">No notification channels</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Add a channel to receive alerts when deploys fail or health checks
              fail
            </p>
            <CreateChannelDialog onSuccess={() => {}} />
          </div>
        ) : (
          <div className="space-y-4">
            {channels.map((channel) => (
              <ChannelCard key={channel.id} channel={channel} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
