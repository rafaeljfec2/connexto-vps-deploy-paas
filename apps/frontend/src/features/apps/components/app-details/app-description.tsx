import { ExternalLink, Folder, GitBranch } from "lucide-react";
import { IconText } from "@/components/icon-text";
import { formatRepositoryUrl } from "@/lib/utils";
import type { App } from "@/types";

interface AppDescriptionProps {
  readonly app: App;
}

export function AppDescription({ app }: AppDescriptionProps) {
  const showWorkdir = app.workdir && app.workdir !== ".";
  return (
    <div className="flex items-center gap-2 sm:gap-4 text-xs sm:text-sm text-muted-foreground flex-wrap">
      <IconText icon={GitBranch} as="span">
        {app.branch}
      </IconText>
      <a
        href={app.repositoryUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center gap-1 hover:text-foreground truncate max-w-[200px] sm:max-w-none"
      >
        <ExternalLink className="h-3.5 w-3.5 sm:h-4 sm:w-4 shrink-0" />
        <span className="truncate">
          {formatRepositoryUrl(app.repositoryUrl)}
        </span>
      </a>
      {showWorkdir && (
        <IconText icon={Folder} as="span">
          <span className="font-mono text-xs truncate max-w-[100px] sm:max-w-none">
            {app.workdir}
          </span>
        </IconText>
      )}
    </div>
  );
}
