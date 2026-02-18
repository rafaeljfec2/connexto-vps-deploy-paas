import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, Loader2, Plus, Settings2, Trash2 } from "lucide-react";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/services/api";
import type {
  CreateNotificationChannelInput,
  CreateNotificationRuleInput,
  NotificationChannel,
  NotificationChannelType,
  NotificationEventType,
  NotificationRule,
} from "@/types";
import { DiscordChannelForm } from "./discord-channel-form";
import { EmailChannelForm } from "./email-channel-form";
import { SlackChannelForm } from "./slack-channel-form";

const CHANNEL_TYPES: { value: NotificationChannelType; label: string }[] = [
  { value: "slack", label: "Slack" },
  { value: "discord", label: "Discord" },
  { value: "email", label: "Email" },
];

const EVENT_TYPES: { value: NotificationEventType; label: string }[] = [
  { value: "deploy_running", label: "Deploy started" },
  { value: "deploy_success", label: "Deploy success" },
  { value: "deploy_failed", label: "Deploy failed" },
  { value: "container_down", label: "Container down" },
  { value: "health_unhealthy", label: "Health unhealthy" },
];

function getEventTypeLabel(eventType: string): string {
  const found = EVENT_TYPES.find((e) => e.value === eventType);
  return found?.label ?? eventType;
}

function ChannelConfigFields({
  type,
  config,
  onChange,
}: {
  readonly type: NotificationChannelType;
  readonly config: Record<string, string>;
  readonly onChange: (config: Record<string, string>) => void;
}) {
  if (type === "slack") {
    return <SlackChannelForm config={config} onChange={onChange} />;
  }
  if (type === "discord") {
    return <DiscordChannelForm config={config} onChange={onChange} />;
  }
  if (type === "email") {
    return <EmailChannelForm config={config} onChange={onChange} />;
  }
  return null;
}

function CreateChannelDialog({
  onSuccess,
}: {
  readonly onSuccess: () => void;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [type, setType] = useState<NotificationChannelType>("slack");
  const [name, setName] = useState("");
  const [config, setConfig] = useState<Record<string, string>>({});

  const createMutation = useMutation({
    mutationFn: (input: CreateNotificationChannelInput) =>
      api.notifications.channels.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
      setOpen(false);
      setName("");
      setConfig({});
      onSuccess();
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const configObj: Record<string, unknown> = {};
    if (type === "slack" || type === "discord") {
      configObj.webhookUrl = config.webhookUrl ?? "";
    } else if (type === "email") {
      configObj.smtpHost = config.smtpHost ?? "";
      configObj.smtpPort = Number.parseInt(config.smtpPort ?? "587", 10) || 587;
      configObj.from = config.from ?? "";
      configObj.to = config.to ?? "";
      if (config.username) configObj.username = config.username;
      if (config.password) configObj.password = config.password;
    }
    createMutation.mutate({ type, name, config: configObj });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Add channel
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add notification channel</DialogTitle>
          <DialogDescription>
            Configure a channel to receive deploy and health alerts
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label
              htmlFor="channelType"
              className="text-sm font-medium leading-none"
            >
              Type
            </label>
            <Select
              value={type}
              onValueChange={(v) => setType(v as NotificationChannelType)}
            >
              <SelectTrigger id="channelType">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CHANNEL_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <label
              htmlFor="channelName"
              className="text-sm font-medium leading-none"
            >
              Name
            </label>
            <Input
              id="channelName"
              placeholder="e.g. Team Slack"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>
          <ChannelConfigFields
            type={type}
            config={config}
            onChange={setConfig}
          />
          {createMutation.isError && (
            <p className="text-sm text-destructive">
              {createMutation.error instanceof Error
                ? createMutation.error.message
                : "Failed to create channel"}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              )}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function AddRuleDialog({
  channel,
  onSuccess,
}: {
  readonly channel: NotificationChannel;
  readonly onSuccess: () => void;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [eventType, setEventType] =
    useState<NotificationEventType>("deploy_failed");
  const [enabled, setEnabled] = useState(true);

  const createMutation = useMutation({
    mutationFn: (input: CreateNotificationRuleInput) =>
      api.notifications.rules.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["notification-rules", channel.id],
      });
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
      setOpen(false);
      onSuccess();
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate({
      eventType,
      channelId: channel.id,
      enabled,
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Add rule
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add notification rule</DialogTitle>
          <DialogDescription>
            Choose which events trigger notifications for {channel.name}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label
              htmlFor="eventType"
              className="text-sm font-medium leading-none"
            >
              Event type
            </label>
            <Select
              value={eventType}
              onValueChange={(v) => setEventType(v as NotificationEventType)}
            >
              <SelectTrigger id="eventType">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {EVENT_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="enabled"
              checked={enabled}
              onCheckedChange={(c) => setEnabled(c === true)}
            />
            <label
              htmlFor="enabled"
              className="text-sm font-normal cursor-pointer"
            >
              Enable rule
            </label>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-destructive">
              {createMutation.error instanceof Error
                ? createMutation.error.message
                : "Failed to create rule"}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              )}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function ChannelCard({ channel }: { readonly channel: NotificationChannel }) {
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
