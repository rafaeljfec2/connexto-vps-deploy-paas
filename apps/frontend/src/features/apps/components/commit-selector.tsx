import { formatDistanceToNow } from "date-fns";
import { ptBR } from "date-fns/locale";
import { ExternalLink, GitCommit, Loader2, Rocket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import type { CommitInfo } from "@/types";
import { useCommits } from "../hooks/use-apps";

interface CommitSelectorProps {
  readonly appId: string;
  readonly onSelect: (sha: string) => void;
  readonly disabled?: boolean;
}

function CommitItem({
  commit,
  onDeploy,
  disabled,
}: {
  readonly commit: CommitInfo;
  readonly onDeploy: (sha: string) => void;
  readonly disabled?: boolean;
}) {
  const firstLine = (commit.message.split("\n")[0] ?? "").slice(0, 50);
  const hasMore = commit.message.length > 50 || commit.message.includes("\n");

  return (
    <div
      className={cn(
        "p-2 rounded-md border border-transparent",
        "hover:bg-muted/50 hover:border-muted-foreground/20 transition-colors",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5 min-w-0">
          <GitCommit className="h-3 w-3 text-muted-foreground shrink-0" />
          <code className="text-[11px] font-mono text-primary bg-primary/10 px-1 py-0.5 rounded">
            {commit.sha.slice(0, 7)}
          </code>
          <span className="text-[11px] text-muted-foreground">
            {formatDistanceToNow(new Date(commit.date), {
              addSuffix: true,
              locale: ptBR,
            })}
          </span>
        </div>
        <div className="flex items-center shrink-0">
          <a
            href={commit.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground hover:text-foreground p-1 rounded hover:bg-muted"
            title="View on GitHub"
          >
            <ExternalLink className="h-3 w-3" />
          </a>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onDeploy(commit.sha)}
            disabled={disabled}
            className="h-5 px-1.5 text-[10px]"
          >
            <Rocket className="h-2.5 w-2.5 mr-0.5" />
            Deploy
          </Button>
        </div>
      </div>
      <p className="text-[11px] mt-1 truncate text-muted-foreground pl-4">
        {firstLine}
        {hasMore && "..."}
      </p>
    </div>
  );
}

export function CommitSelectorInline({
  appId,
  onSelect,
  disabled,
}: CommitSelectorProps) {
  const { data: commits, isLoading, error } = useCommits(appId, 30);

  return (
    <div>
      {isLoading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-sm text-muted-foreground">
            Loading commits...
          </span>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-destructive">
          <p className="text-sm">Failed to load commits</p>
          <p className="text-xs text-muted-foreground mt-1">
            Check if the GitHub token is configured
          </p>
        </div>
      )}

      {commits && commits.length === 0 && (
        <div className="text-center py-8 text-muted-foreground">
          <GitCommit className="h-6 w-6 mx-auto mb-2" />
          <p className="text-sm">No commits found</p>
        </div>
      )}

      {commits && commits.length > 0 && (
        <ScrollArea className="h-[450px] w-full">
          <div className="space-y-1 pr-3 max-w-full">
            {commits.map((commit) => (
              <CommitItem
                key={commit.sha}
                commit={commit}
                onDeploy={onSelect}
                disabled={disabled}
              />
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}
