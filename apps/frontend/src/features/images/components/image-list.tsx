import { useState } from "react";
import { HardDrive, Loader2, Search, Trash2 } from "lucide-react";
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
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { useImages, usePruneImages, useRemoveImage } from "../hooks/use-images";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${Number.parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function ImageList() {
  const [search, setSearch] = useState("");
  const [showDanglingOnly, setShowDanglingOnly] = useState(false);
  const [showPruneDialog, setShowPruneDialog] = useState(false);
  const [imageToDelete, setImageToDelete] = useState<string | null>(null);

  const { data: images, isLoading, error } = useImages();
  const removeImage = useRemoveImage();
  const pruneImages = usePruneImages();

  const filteredImages = images?.filter((image) => {
    const matchesSearch =
      search === "" ||
      image.repository.toLowerCase().includes(search.toLowerCase()) ||
      image.tag.toLowerCase().includes(search.toLowerCase()) ||
      image.id.toLowerCase().includes(search.toLowerCase());

    const matchesDangling = !showDanglingOnly || image.dangling;

    return matchesSearch && matchesDangling;
  });

  const danglingCount = images?.filter((img) => img.dangling).length ?? 0;
  const totalSize = images?.reduce((sum, img) => sum + img.size, 0) ?? 0;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex gap-4">
          <Skeleton className="h-10 w-[300px]" />
          <Skeleton className="h-10 w-[120px]" />
        </div>
        <Card>
          <div className="p-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton
                key={`image-skeleton-${i.toString()}`}
                className="h-16 w-full"
              />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage message="Failed to load images" />;
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search images..."
            className="pl-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant={showDanglingOnly ? "default" : "outline"}
            size="sm"
            onClick={() => setShowDanglingOnly(!showDanglingOnly)}
          >
            Dangling Only ({danglingCount})
          </Button>
          {danglingCount > 0 && (
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setShowPruneDialog(true)}
              disabled={pruneImages.isPending}
            >
              {pruneImages.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Trash2 className="h-4 w-4 mr-2" />
              )}
              Prune All
            </Button>
          )}
        </div>
      </div>

      <div className="flex gap-4 text-sm text-muted-foreground">
        <span>{images?.length ?? 0} total images</span>
        <span>•</span>
        <span>{formatBytes(totalSize)} total size</span>
        {danglingCount > 0 && (
          <>
            <span>•</span>
            <span className="text-orange-500">{danglingCount} dangling</span>
          </>
        )}
      </div>

      {filteredImages?.length === 0 ? (
        <EmptyState
          icon={HardDrive}
          title="No images found"
          description={
            search || showDanglingOnly
              ? "Try adjusting your filters or search query."
              : "No Docker images found on this server."
          }
        />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground min-w-[480px]">
                    Repository
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground">
                    Tag
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground hidden md:table-cell">
                    ID
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground hidden lg:table-cell">
                    Size
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground hidden md:table-cell">
                    Created
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-muted-foreground w-10"></th>
                </tr>
              </thead>
              <tbody>
                {filteredImages?.map((image) => (
                  <tr
                    key={image.id}
                    className="border-b border-border hover:bg-muted/50 transition-colors"
                  >
                    <td className="py-3 px-4 min-w-0">
                      <div className="flex items-center gap-2 min-w-0">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="font-medium truncate block min-w-0">
                                {image.repository === "<none>" ? (
                                  <span className="text-muted-foreground italic">
                                    none
                                  </span>
                                ) : (
                                  image.repository
                                )}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-md">
                              <p className="font-mono text-xs break-all">
                                {image.repository}:{image.tag}
                              </p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                        {image.dangling && (
                          <Badge
                            variant="outline"
                            className="text-orange-500 border-orange-500"
                          >
                            dangling
                          </Badge>
                        )}
                        {image.containers > 0 ? (
                          <Badge variant="secondary" className="text-xs">
                            In use ({image.containers})
                          </Badge>
                        ) : (
                          <Badge
                            variant="outline"
                            className="text-xs text-muted-foreground"
                          >
                            Unused
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <span className="text-xs">
                        {image.tag === "<none>" ? (
                          <span className="text-muted-foreground italic">
                            none
                          </span>
                        ) : (
                          image.tag
                        )}
                      </span>
                    </td>
                    <td className="py-3 px-4 hidden md:table-cell">
                      <span className="text-xs text-muted-foreground font-mono">
                        {image.id.slice(0, 12)}
                      </span>
                    </td>
                    <td className="py-3 px-4 hidden lg:table-cell whitespace-nowrap">
                      <span className="text-xs text-muted-foreground">
                        {formatBytes(image.size)}
                      </span>
                    </td>
                    <td className="py-3 px-4 hidden md:table-cell whitespace-nowrap">
                      <span className="text-xs text-muted-foreground">
                        {formatDate(image.created)}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setImageToDelete(image.id)}
                        disabled={removeImage.isPending}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <AlertDialog open={showPruneDialog} onOpenChange={setShowPruneDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Prune all dangling images?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove all dangling images ({danglingCount} images) that
              are not being used by any container. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pruneImages.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                pruneImages.mutate(undefined, {
                  onSuccess: () => setShowPruneDialog(false),
                });
              }}
              disabled={pruneImages.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {pruneImages.isPending ? "Pruning..." : "Prune All"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!imageToDelete}
        onOpenChange={() => setImageToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove image?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the image. If the image is being used by a
              container, you may need to force removal.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removeImage.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (imageToDelete) {
                  removeImage.mutate(
                    { id: imageToDelete, force: true },
                    { onSuccess: () => setImageToDelete(null) },
                  );
                }
              }}
              disabled={removeImage.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeImage.isPending ? "Removing..." : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
