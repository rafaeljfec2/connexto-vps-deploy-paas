import { useState } from "react";
import {
  Calendar,
  FolderOpen,
  HardDrive,
  Loader2,
  Plus,
  Search,
  Trash2,
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
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  useCreateVolume,
  useRemoveVolume,
  useVolumes,
} from "../hooks/use-volumes";

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface VolumesManagerProps {
  readonly containerVolumes?: readonly string[];
}

export function VolumesManager({ containerVolumes = [] }: VolumesManagerProps) {
  const { data: volumes, isLoading, error } = useVolumes();
  const createVolume = useCreateVolume();
  const removeVolume = useRemoveVolume();

  const [searchQuery, setSearchQuery] = useState("");
  const [newVolumeName, setNewVolumeName] = useState("");
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [volumeToDelete, setVolumeToDelete] = useState<string | null>(null);

  const isScoped = containerVolumes !== undefined;
  const scopedVolumes = isScoped
    ? volumes?.filter((vol) => containerVolumes.includes(vol.name))
    : volumes;
  const filteredVolumes = scopedVolumes?.filter((vol) =>
    (vol.name ?? "").toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const handleCreate = async () => {
    if (!newVolumeName.trim()) return;
    await createVolume.mutateAsync(newVolumeName.trim());
    setNewVolumeName("");
    setShowCreateDialog(false);
  };

  const handleDelete = async () => {
    if (!volumeToDelete) return;
    await removeVolume.mutateAsync(volumeToDelete);
    setVolumeToDelete(null);
  };

  const isVolumeInUse = (volumeName: string) =>
    containerVolumes?.some((cv) => cv.includes(volumeName)) ?? false;

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
        Failed to load volumes: {error.message}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search volumes..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button size="sm" onClick={() => setShowCreateDialog(true)}>
          <Plus className="h-4 w-4 mr-1" />
          Create
        </Button>
      </div>

      {containerVolumes.length > 0 && (
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">
            Container Volumes
          </p>
          <div className="flex flex-wrap gap-2">
            {containerVolumes.map((volPath) => (
              <Badge
                key={volPath}
                variant="secondary"
                className="flex items-center gap-1"
              >
                <HardDrive className="h-3 w-3" />
                <span className="font-mono text-xs truncate max-w-[200px]">
                  {volPath}
                </span>
              </Badge>
            ))}
          </div>
        </div>
      )}

      <div className="border rounded-lg divide-y">
        {filteredVolumes?.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            <HardDrive className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No volumes found</p>
          </div>
        ) : (
          filteredVolumes?.map((volume) => (
            <div
              key={volume.name}
              className="flex items-center justify-between p-3 hover:bg-muted/50 transition-colors"
            >
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <HardDrive className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="font-medium truncate">{volume.name}</p>
                    {isVolumeInUse(volume.name) && (
                      <Badge variant="outline" className="text-xs h-5">
                        In use
                      </Badge>
                    )}
                  </div>
                  <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-3 text-xs text-muted-foreground mt-1">
                    <span className="flex items-center gap-1">
                      <span className="font-medium">Driver:</span>
                      {volume.driver}
                    </span>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="flex items-center gap-1 truncate cursor-help">
                            <FolderOpen className="h-3 w-3 shrink-0" />
                            <span className="truncate font-mono max-w-[250px]">
                              {volume.mountpoint}
                            </span>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="font-mono text-xs">
                            {volume.mountpoint}
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                    {volume.createdAt && (
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDate(volume.createdAt)}
                      </span>
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
                      onClick={() => setVolumeToDelete(volume.name)}
                      disabled={
                        removeVolume.isPending || isVolumeInUse(volume.name)
                      }
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {isVolumeInUse(volume.name)
                      ? "Cannot delete: volume is in use"
                      : "Delete volume"}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          ))
        )}
      </div>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Volume</DialogTitle>
            <DialogDescription>
              Create a new Docker volume for persistent data storage.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              placeholder="Volume name"
              value={newVolumeName}
              onChange={(e) => setNewVolumeName(e.target.value)}
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
              disabled={!newVolumeName.trim() || createVolume.isPending}
            >
              {createVolume.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!volumeToDelete}
        onOpenChange={(open) => !open && setVolumeToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Volume</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the volume &quot;{volumeToDelete}
              &quot;? All data stored in this volume will be permanently lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeVolume.isPending && (
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
