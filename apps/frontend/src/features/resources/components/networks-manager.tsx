import { useState } from "react";
import {
  Loader2,
  Network,
  Plug,
  Plus,
  Search,
  Server,
  Trash2,
  Unplug,
} from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
import { Input } from "@/components/ui/input";
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
  useCreateNetwork,
  useDisconnectContainerFromNetwork,
  useNetworks,
  useRemoveNetwork,
} from "../hooks/use-networks";

interface NetworksManagerProps {
  readonly containerId?: string;
  readonly containerNetworks?: readonly string[];
}

function getDeleteTooltip(
  hasContainers: boolean,
  isSystemNetwork: boolean,
): string {
  if (hasContainers) {
    return "Cannot delete: network has containers";
  }
  if (isSystemNetwork) {
    return "Cannot delete: system network";
  }
  return "Delete network";
}

export function NetworksManager({
  containerId,
  containerNetworks = [],
}: NetworksManagerProps) {
  const { data: networks, isLoading, error } = useNetworks();
  const createNetwork = useCreateNetwork();
  const removeNetwork = useRemoveNetwork();
  const connectToNetwork = useConnectContainerToNetwork();
  const disconnectFromNetwork = useDisconnectContainerFromNetwork();

  const [searchQuery, setSearchQuery] = useState("");
  const [newNetworkName, setNewNetworkName] = useState("");
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [networkToDelete, setNetworkToDelete] = useState<string | null>(null);
  const [showConnectDialog, setShowConnectDialog] = useState(false);
  const [selectedNetwork, setSelectedNetwork] = useState<string>("");

  const scopedNetworks = containerId
    ? networks?.filter((net) => containerNetworks.includes(net.name))
    : networks;
  const filteredNetworks = scopedNetworks?.filter((net) =>
    (net.name ?? "").toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const availableNetworksToConnect = networks?.filter(
    (net) => !containerNetworks.includes(net.name),
  );

  const handleCreate = async () => {
    if (!newNetworkName.trim()) return;
    await createNetwork.mutateAsync(newNetworkName.trim());
    setNewNetworkName("");
    setShowCreateDialog(false);
  };

  const handleDelete = async () => {
    if (!networkToDelete) return;
    await removeNetwork.mutateAsync(networkToDelete);
    setNetworkToDelete(null);
  };

  const handleConnect = async () => {
    if (!containerId || !selectedNetwork) return;
    await connectToNetwork.mutateAsync({
      containerId,
      network: selectedNetwork,
    });
    setSelectedNetwork("");
    setShowConnectDialog(false);
  };

  const handleDisconnect = async (network: string) => {
    if (!containerId) return;
    await disconnectFromNetwork.mutateAsync({
      containerId,
      network,
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-8 text-destructive">
        Failed to load networks: {error.message}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search networks..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex gap-2">
          {containerId && availableNetworksToConnect?.length ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowConnectDialog(true)}
            >
              <Plug className="h-4 w-4 mr-1" />
              Connect
            </Button>
          ) : null}
          <Button size="sm" onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Create
          </Button>
        </div>
      </div>

      {containerId && containerNetworks.length > 0 && (
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">
            Connected Networks
          </p>
          <div className="flex flex-wrap gap-2">
            {containerNetworks.map((netName) => (
              <Badge
                key={netName}
                variant="secondary"
                className="flex items-center gap-1 pr-1"
              >
                <Network className="h-3 w-3" />
                {netName}
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-5 w-5 p-0 ml-1 hover:bg-destructive/20"
                        onClick={() => handleDisconnect(netName)}
                        disabled={disconnectFromNetwork.isPending}
                      >
                        <Unplug className="h-3 w-3" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Disconnect from network</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </Badge>
            ))}
          </div>
        </div>
      )}

      <div className="border rounded-lg divide-y">
        {filteredNetworks?.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            <Network className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No networks found</p>
          </div>
        ) : (
          filteredNetworks?.map((network) => {
            const networkContainers = network.containers ?? [];
            return (
              <div
                key={network.id}
                className="flex items-center justify-between p-3 hover:bg-muted/50 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0 flex-1">
                  <Network className="h-4 w-4 text-muted-foreground shrink-0" />
                  <div className="min-w-0">
                    <p className="font-medium truncate">{network.name}</p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{network.driver}</span>
                      <span>•</span>
                      <span>{network.scope}</span>
                      {networkContainers.length > 0 && (
                        <>
                          <span>•</span>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="flex items-center gap-1 cursor-help">
                                  <Server className="h-3 w-3" />
                                  {networkContainers.length}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="font-medium mb-1">Containers:</p>
                                <ul className="text-xs">
                                  {networkContainers.map((c) => (
                                    <li key={c}>{c}</li>
                                  ))}
                                </ul>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </>
                      )}
                      {network.internal && (
                        <Badge variant="outline" className="text-xs h-5">
                          Internal
                        </Badge>
                      )}
                    </div>
                  </div>
                </div>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => setNetworkToDelete(network.name)}
                        disabled={
                          removeNetwork.isPending ||
                          networkContainers.length > 0 ||
                          ["bridge", "host", "none"].includes(network.name)
                        }
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {getDeleteTooltip(
                        networkContainers.length > 0,
                        ["bridge", "host", "none"].includes(network.name),
                      )}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            );
          })
        )}
      </div>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Network</DialogTitle>
            <DialogDescription>
              Create a new Docker network for container communication.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              placeholder="Network name"
              value={newNetworkName}
              onChange={(e) => setNewNetworkName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCreateDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!newNetworkName.trim() || createNetwork.isPending}
            >
              {createNetwork.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showConnectDialog} onOpenChange={setShowConnectDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Connect to Network</DialogTitle>
            <DialogDescription>
              Connect this container to an existing network.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Select value={selectedNetwork} onValueChange={setSelectedNetwork}>
              <SelectTrigger>
                <SelectValue placeholder="Select a network" />
              </SelectTrigger>
              <SelectContent>
                {availableNetworksToConnect?.map((net) => (
                  <SelectItem key={net.id} value={net.name}>
                    {net.name} ({net.driver})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
              disabled={!selectedNetwork || connectToNetwork.isPending}
            >
              {connectToNetwork.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Connect
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!networkToDelete}
        onOpenChange={(open) => !open && setNetworkToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Network</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the network &quot;
              {networkToDelete}
              &quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeNetwork.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
