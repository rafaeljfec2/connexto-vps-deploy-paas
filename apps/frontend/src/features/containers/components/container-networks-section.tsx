import { useMemo, useState } from "react";
import { Loader2, Network, Plug, Unplug } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  useConnectContainerToNetwork,
  useDisconnectContainerFromNetwork,
  useNetworks,
} from "@/features/resources/hooks/use-networks";

interface ContainerNetworksSectionProps {
  readonly containerId: string;
  readonly containerNetworks: readonly string[];
  readonly serverId?: string;
}

export function ContainerNetworksSection({
  containerId,
  containerNetworks,
  serverId,
}: ContainerNetworksSectionProps) {
  const { data: networks, isLoading } = useNetworks(serverId);
  const connectMutation = useConnectContainerToNetwork();
  const disconnectMutation = useDisconnectContainerFromNetwork();

  const [showConnectDialog, setShowConnectDialog] = useState(false);
  const [selectedNetwork, setSelectedNetwork] = useState<string>("");

  const availableNetworks = useMemo(() => {
    if (!networks) return [];
    return networks.filter((net) => !containerNetworks.includes(net.name));
  }, [networks, containerNetworks]);

  const handleConnect = async () => {
    if (!selectedNetwork) return;
    await connectMutation.mutateAsync({
      containerId,
      network: selectedNetwork,
      serverId,
    });
    setSelectedNetwork("");
    setShowConnectDialog(false);
  };

  const handleDisconnect = (network: string) => {
    disconnectMutation.mutate({ containerId, network, serverId });
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <h4 className="text-sm font-medium flex items-center gap-2">
          <Network className="h-4 w-4" />
          Networks ({containerNetworks.length})
        </h4>
        <Button
          variant="outline"
          size="sm"
          className="h-7 text-xs"
          onClick={() => setShowConnectDialog(true)}
          disabled={isLoading || availableNetworks.length === 0}
        >
          <Plug className="h-3 w-3 mr-1" />
          Connect
        </Button>
      </div>

      <div className="flex flex-wrap gap-1.5">
        {containerNetworks.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            Not connected to any networks.
          </p>
        ) : (
          containerNetworks.map((netName) => {
            const isPending =
              disconnectMutation.isPending &&
              disconnectMutation.variables?.containerId === containerId &&
              disconnectMutation.variables?.network === netName;
            return (
              <Badge
                key={netName}
                variant="secondary"
                className="flex items-center gap-1 pr-0.5 text-xs"
              >
                <Network className="h-3 w-3" />
                {netName}
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-5 w-5 p-0 ml-0.5 hover:bg-destructive/20"
                        onClick={() => handleDisconnect(netName)}
                        disabled={isPending}
                        aria-label={`Disconnect from ${netName}`}
                      >
                        {isPending ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <Unplug className="h-3 w-3" />
                        )}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Disconnect</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </Badge>
            );
          })
        )}
      </div>

      <Dialog open={showConnectDialog} onOpenChange={setShowConnectDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Connect to network</DialogTitle>
            <DialogDescription>
              Connect this container to an existing Docker network.
            </DialogDescription>
          </DialogHeader>
          <div className="py-2">
            <Select
              value={selectedNetwork}
              onValueChange={setSelectedNetwork}
              disabled={availableNetworks.length === 0}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a network" />
              </SelectTrigger>
              <SelectContent>
                {availableNetworks.map((net) => (
                  <SelectItem key={net.id} value={net.name}>
                    {net.name} ({net.driver})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {availableNetworks.length === 0 && (
              <p className="text-xs text-muted-foreground mt-2">
                No additional networks available.
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowConnectDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleConnect}
              disabled={!selectedNetwork || connectMutation.isPending}
            >
              {connectMutation.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Connect
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
