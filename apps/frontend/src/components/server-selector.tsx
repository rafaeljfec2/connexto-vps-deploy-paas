import { useEffect } from "react";
import { useAuth } from "@/contexts/auth-context";
import { Monitor, Server } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useServers } from "@/features/servers/hooks/use-servers";

const LOCAL_SERVER_VALUE = "__local__";

interface ServerSelectorProps {
  readonly value: string | undefined;
  readonly onChange: (serverId: string | undefined) => void;
}

export function ServerSelector({ value, onChange }: ServerSelectorProps) {
  const { data: servers } = useServers();
  const { isAdmin } = useAuth();

  useEffect(() => {
    if (!isAdmin && !value && servers?.[0]) {
      onChange(servers[0].id);
    }
  }, [isAdmin, value, servers, onChange]);

  if (!servers?.length) return null;

  return (
    <Select
      value={value ?? LOCAL_SERVER_VALUE}
      onValueChange={(v) => onChange(v === LOCAL_SERVER_VALUE ? undefined : v)}
    >
      <SelectTrigger className="w-[200px]">
        <SelectValue placeholder="Select server" />
      </SelectTrigger>
      <SelectContent>
        {isAdmin && (
          <SelectItem value={LOCAL_SERVER_VALUE}>
            <div className="flex items-center gap-2">
              <Monitor className="h-3.5 w-3.5" />
              <span>Local Server</span>
            </div>
          </SelectItem>
        )}
        {servers.map((server) => (
          <SelectItem key={server.id} value={server.id}>
            <div className="flex items-center gap-2">
              <Server className="h-3.5 w-3.5" />
              <span>{server.name}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
