import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, Loader2, Settings2, Trash2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { api } from "@/services/api";
import type { NotificationChannel, NotificationRule } from "@/types";
import { AddRuleDialog } from "./add-rule-dialog";
import { CHANNEL_TYPES, getEventTypeLabel } from "./notification-constants";

interface ChannelCardProps {
  readonly channel: NotificationChannel;
}

export function ChannelCard({ channel }: ChannelCardProps) {
  const queryClient = useQueryClient();
  const [expanded, setExpanded] = useState(false);

  const { data: rules = [], isLoading: rulesLoading } = useQuery({
    queryKey: ["notification-rules", channel.id],
    queryFn: () => api.notifications.channels.rules(channel.id),
    enabled: expanded,
  });

  const deleteChannelMutation = useMutation({
    mutationFn: () => api.notifications.channels.delete(channel.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
    },
  });

  const deleteRuleMutation = useMutation({
    mutationFn: (ruleId: string) => api.notifications.rules.delete(ruleId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["notification-rules", channel.id],
      });
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
    },
  });

  const updateRuleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.notifications.rules.update(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["notification-rules", channel.id],
      });
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
    },
  });

  const typeLabel =
    CHANNEL_TYPES.find((t) => t.value === channel.type)?.label ?? channel.type;

  return (
    <Card>
      <CardHeader
        className="cursor-pointer py-4 sm:py-6"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted shrink-0">
              <Bell className="h-5 w-5 text-muted-foreground" />
            </div>
            <div className="min-w-0">
              <CardTitle className="text-base sm:text-lg">
                {channel.name}
              </CardTitle>
              <CardDescription className="flex items-center gap-2 mt-1">
                <Badge variant="secondary" className="text-xs">
                  {typeLabel}
                </Badge>
                {channel.appId && <span className="text-xs">App-specific</span>}
              </CardDescription>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <AddRuleDialog channel={channel} onSuccess={() => {}} />
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive"
                  onClick={(e) => e.stopPropagation()}
                  disabled={deleteChannelMutation.isPending}
                >
                  {deleteChannelMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete channel</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to delete "{channel.name}"? All
                    associated rules will be removed.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => deleteChannelMutation.mutate()}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
            <Button
              variant="ghost"
              size="icon"
              onClick={(e) => {
                e.stopPropagation();
                setExpanded(!expanded);
              }}
            >
              <Settings2
                className={`h-4 w-4 transition-transform ${expanded ? "rotate-90" : ""}`}
              />
            </Button>
          </div>
        </div>
      </CardHeader>
      {expanded && (
        <CardContent className="pt-0 border-t">
          <div className="pt-4 space-y-3">
            <h4 className="text-sm font-medium">Rules</h4>
            {rulesLoading && (
              <div className="flex justify-center py-4">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            )}
            {!rulesLoading && rules.length === 0 && (
              <p className="text-sm text-muted-foreground">
                No rules configured. Add a rule to receive notifications.
              </p>
            )}
            {!rulesLoading && rules.length > 0 && (
              <ul className="space-y-2">
                {rules.map((rule: NotificationRule) => (
                  <li
                    key={rule.id}
                    className="flex items-center justify-between gap-2 py-2 px-3 rounded-md bg-muted/50"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Checkbox
                        checked={rule.enabled}
                        onCheckedChange={(c) =>
                          updateRuleMutation.mutate({
                            id: rule.id,
                            enabled: c === true,
                          })
                        }
                        disabled={updateRuleMutation.isPending}
                      />
                      <span className="text-sm truncate">
                        {getEventTypeLabel(rule.eventType)}
                      </span>
                    </div>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive hover:text-destructive shrink-0"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete rule</AlertDialogTitle>
                          <AlertDialogDescription>
                            Remove this notification rule?
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => deleteRuleMutation.mutate(rule.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  );
}
