import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/services/api";
import type {
  CreateNotificationRuleInput,
  NotificationChannel,
  NotificationEventType,
} from "@/types";
import { EVENT_TYPES } from "./notification-constants";

interface AddRuleDialogProps {
  readonly channel: NotificationChannel;
  readonly onSuccess: () => void;
}

export function AddRuleDialog({ channel, onSuccess }: AddRuleDialogProps) {
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
