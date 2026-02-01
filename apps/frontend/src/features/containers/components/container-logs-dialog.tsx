import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useContainerLogs } from "../hooks/use-containers";

interface ContainerLogsDialogProps {
  readonly containerId: string | null;
  readonly containerName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function ContainerLogsDialog({
  containerId,
  containerName,
  open,
  onOpenChange,
}: ContainerLogsDialogProps) {
  const { data: logsData } = useContainerLogs(
    open ? (containerId ?? undefined) : undefined,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>Logs - {containerName}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="h-[60vh] rounded-md border bg-muted/30 p-4">
          <pre className="text-xs font-mono whitespace-pre-wrap">
            {logsData?.logs ?? "Loading logs..."}
          </pre>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
