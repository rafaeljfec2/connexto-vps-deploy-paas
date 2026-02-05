import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { TerminalView } from "@/components/terminal";
import { api } from "@/services/api";

interface ContainerConsoleDialogProps {
  readonly containerId: string | null;
  readonly containerName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function ContainerConsoleDialog({
  containerId,
  containerName,
  open,
  onOpenChange,
}: ContainerConsoleDialogProps) {
  const wsUrl = containerId ? api.containers.consoleUrl(containerId) : "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl h-[80vh] flex flex-col p-0">
        <DialogHeader className="px-6 pt-6 pb-2">
          <DialogTitle>Console - {containerName}</DialogTitle>
        </DialogHeader>
        {containerId && (
          <TerminalView
            wsUrl={wsUrl}
            autoConnect={open}
            className="flex-1 min-h-[240px] p-4"
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
