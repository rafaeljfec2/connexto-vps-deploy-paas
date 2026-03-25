import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
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
  NotificationChannelType,
} from "@/types";
import { ChannelConfigFields } from "./channel-config-fields";
import { CHANNEL_TYPES } from "./notification-constants";

interface CreateChannelDialogProps {
  readonly onSuccess: () => void;
}

export function CreateChannelDialog({ onSuccess }: CreateChannelDialogProps) {
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
