import { useEffect, useMemo } from "react";
import { useAuth } from "@/contexts/auth-context";
import { Check, Loader2, Monitor, Server, Wifi, WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { useServers } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";
import type { Server as ServerType } from "@/types";
import type { StepProps } from "./types";

const LOCAL_SERVER_ID = "";

interface ServerOptionProps {
  readonly selected: boolean;
  readonly onSelect: () => void;
  readonly icon: React.ReactNode;
  readonly title: string;
  readonly description: string;
  readonly badge?: React.ReactNode;
  readonly disabled?: boolean;
}

function ServerOption({
  selected,
  onSelect,
  icon,
  title,
  description,
  badge,
  disabled,
}: ServerOptionProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onSelect}
      className={cn(
        "w-full flex items-start gap-3 p-3 sm:p-4 border rounded-lg text-left transition-colors",
        selected
          ? "border-primary bg-primary/5 ring-1 ring-primary"
          : "hover:border-muted-foreground/40",
        disabled && "opacity-50 cursor-not-allowed",
      )}
    >
      <div
        className={cn(
          "mt-0.5 shrink-0",
          selected ? "text-primary" : "text-muted-foreground",
        )}
      >
        {icon}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium truncate">{title}</p>
          {badge}
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>
      {selected && <Check className="h-4 w-4 text-primary mt-0.5 shrink-0" />}
    </button>
  );
}

function ServerStatusBadge({
  status,
}: {
  readonly status: ServerType["status"];
}) {
  const isOnline = status === "online";

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded-full",
        isOnline
          ? "bg-green-500/10 text-green-600 dark:text-green-400"
          : "bg-muted text-muted-foreground",
      )}
    >
      {isOnline ? (
        <Wifi className="h-2.5 w-2.5" />
      ) : (
        <WifiOff className="h-2.5 w-2.5" />
      )}
      {status}
    </span>
  );
}

export function ServerStep({
  data,
  onUpdate,
  onNext,
  onBack,
}: Readonly<StepProps>) {
  const { data: servers, isLoading } = useServers();
  const { isAdmin } = useAuth();
  const selectedId = data.serverId;

  const handleSelect = (serverId: string) => {
    onUpdate({ serverId });
  };

  const onlineServers = useMemo(
    () => servers?.filter((s) => s.status === "online") ?? [],
    [servers],
  );
  const offlineServers = useMemo(
    () => servers?.filter((s) => s.status !== "online") ?? [],
    [servers],
  );

  useEffect(() => {
    if (
      !isAdmin &&
      selectedId === LOCAL_SERVER_ID &&
      onlineServers.length > 0
    ) {
      onUpdate({ serverId: onlineServers[0].id });
    }
  }, [isAdmin, selectedId, onlineServers, onUpdate]);

  return (
    <Card className="border-0 shadow-none md:border md:shadow-sm">
      <CardContent className="p-0 md:p-6 space-y-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <Server className="h-5 w-5" />
            <h3 className="font-semibold">Choose deploy target</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Select where your application will be deployed.
          </p>
        </div>

        <div className="space-y-3">
          {isAdmin && (
            <ServerOption
              selected={selectedId === LOCAL_SERVER_ID}
              onSelect={() => handleSelect(LOCAL_SERVER_ID)}
              icon={<Monitor className="h-5 w-5" />}
              title="Local Server"
              description="Deploy to this machine (default)"
              badge={
                <span className="inline-flex items-center text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-blue-500/10 text-blue-600 dark:text-blue-400">
                  default
                </span>
              }
            />
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-6 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" />
              <span className="text-sm">Loading servers...</span>
            </div>
          )}

          {onlineServers.map((server) => (
            <ServerOption
              key={server.id}
              selected={selectedId === server.id}
              onSelect={() => handleSelect(server.id)}
              icon={<Server className="h-5 w-5" />}
              title={server.name}
              description={`${server.host}:${server.sshPort}`}
              badge={<ServerStatusBadge status={server.status} />}
            />
          ))}

          {offlineServers.map((server) => (
            <ServerOption
              key={server.id}
              selected={false}
              onSelect={() => {}}
              icon={<Server className="h-5 w-5" />}
              title={server.name}
              description={`${server.host}:${server.sshPort}`}
              badge={<ServerStatusBadge status={server.status} />}
              disabled
            />
          ))}

          {!isLoading && servers?.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-4 border rounded-lg bg-muted/30">
              No remote servers configured. You can add servers later in
              Settings.
            </p>
          )}
        </div>
      </CardContent>

      <CardFooter className="p-0 pt-6 md:p-6 md:pt-0 flex flex-col md:flex-row gap-3">
        <Button
          type="button"
          variant="outline"
          className="w-full md:w-auto"
          onClick={onBack}
        >
          Back
        </Button>
        <Button className="w-full md:w-auto md:ml-auto" onClick={onNext}>
          Continue to Environment
        </Button>
      </CardFooter>
    </Card>
  );
}
