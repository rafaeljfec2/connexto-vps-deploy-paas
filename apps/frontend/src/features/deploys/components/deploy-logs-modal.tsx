import { Terminal } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { truncateCommitSha } from "@/lib/utils";
import type { Deployment } from "@/types";
import { LogViewer } from "./log-viewer";

interface DeployLogsModalProps {
  readonly deployment: Deployment | null | undefined;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function DeployLogsModal({
  deployment,
  open,
  onOpenChange,
}: DeployLogsModalProps) {
  const sha = deployment ? truncateCommitSha(deployment.commitSha) : null;
  const appVersion = deployment?.appVersion?.trim() ?? "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] max-w-5xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="shrink-0 border-b border-border px-4 py-3">
          <DialogTitle className="flex min-w-0 items-center gap-2 text-base">
            <Terminal className="h-4 w-4 shrink-0" />
            <span>Deploy Logs</span>
            {sha && (
              <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs font-normal text-muted-foreground">
                {sha}
              </code>
            )}
            {appVersion !== "" && (
              <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs font-normal text-muted-foreground">
                v{appVersion}
              </code>
            )}
          </DialogTitle>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-hidden p-4">
          <LogViewer
            key={deployment?.id ?? "no-deploy"}
            logs={deployment?.logs ?? null}
            title={sha ? `Logs (${sha})` : "Logs"}
            contentClassName="h-[min(60vh,520px)] border border-slate-800 bg-slate-950"
            hideExpand
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
