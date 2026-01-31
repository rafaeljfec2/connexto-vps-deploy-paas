import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Check,
  ChevronsUpDown,
  GitBranch,
  Lock,
  RefreshCw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import { GitHubConnect } from "@/components/github-connect";
import { cn, filterRepositories } from "@/lib/utils";
import { type GitHubRepository, api } from "@/services/api";

interface RepoSelectorProps {
  readonly value?: string;
  readonly onSelect: (repo: GitHubRepository | null) => void;
  readonly installationId?: string;
}

function RepoSelectorLoading() {
  return (
    <div
      className="space-y-2"
      role="status"
      aria-label="Loading repositories"
      aria-live="polite"
    >
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-4 w-48" />
    </div>
  );
}

interface RepoSelectorErrorProps {
  readonly onRetry: () => void;
}

function RepoSelectorError({ onRetry }: RepoSelectorErrorProps) {
  return (
    <div className="flex flex-col gap-2 p-4 border border-destructive/50 rounded-md bg-destructive/10">
      <p className="text-sm text-destructive">
        Failed to load repositories. Please try again.
      </p>
      <Button
        variant="outline"
        size="sm"
        onClick={onRetry}
        className="w-fit"
        aria-label="Retry loading repositories"
      >
        <RefreshCw className="h-4 w-4 mr-2" />
        Retry
      </Button>
    </div>
  );
}

interface RepoItemProps {
  readonly repo: GitHubRepository;
  readonly isSelected: boolean;
}

function RepoItem({ repo, isSelected }: RepoItemProps) {
  return (
    <>
      <Check
        className={cn("h-4 w-4", isSelected ? "opacity-100" : "opacity-0")}
        aria-hidden="true"
      />
      <div className="flex flex-col flex-1 min-w-0">
        <div className="flex items-center gap-2">
          {repo.private && (
            <Lock
              className="h-3 w-3 text-muted-foreground shrink-0"
              aria-label="Private repository"
            />
          )}
          <span className="font-medium truncate">{repo.fullName}</span>
        </div>
        {repo.description && (
          <span className="text-xs text-muted-foreground truncate">
            {repo.description}
          </span>
        )}
        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-1">
          {repo.language && <span>{repo.language}</span>}
          <span className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" aria-hidden="true" />
            {repo.defaultBranch}
          </span>
        </div>
      </div>
    </>
  );
}

export function RepoSelector({
  value,
  onSelect,
  installationId,
}: RepoSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["github", "repos", installationId],
    queryFn: () => api.github.repos(installationId),
    refetchOnWindowFocus: true,
    staleTime: 30 * 1000,
  });

  const filteredRepos = useMemo(
    () => filterRepositories(data?.repositories ?? [], searchQuery),
    [data?.repositories, searchQuery],
  );

  const selectedRepo = data?.repositories.find(
    (repo) => repo.fullName === value,
  );

  if (isLoading) {
    return <RepoSelectorLoading />;
  }

  if (error) {
    return <RepoSelectorError onRetry={() => refetch()} />;
  }

  if (data?.needInstall) {
    return <GitHubConnect message={data.installMessage} />;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a repository"
          aria-haspopup="listbox"
          className="w-full justify-between"
        >
          {selectedRepo ? (
            <span className="flex items-center gap-2 truncate">
              {selectedRepo.private && (
                <Lock className="h-3 w-3 text-muted-foreground" />
              )}
              {selectedRepo.fullName}
            </span>
          ) : (
            <span className="text-muted-foreground">
              Select a repository...
            </span>
          )}
          <ChevronsUpDown
            className="ml-2 h-4 w-4 shrink-0 opacity-50"
            aria-hidden="true"
          />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[calc(100vw-2rem)] sm:w-[400px] p-0"
        align="start"
      >
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Search repositories..."
            value={searchQuery}
            onValueChange={setSearchQuery}
            aria-label="Search repositories"
          />
          <CommandList>
            <CommandEmpty>No repository found.</CommandEmpty>
            <CommandGroup>
              {filteredRepos.map((repo) => (
                <CommandItem
                  key={repo.id}
                  value={repo.fullName}
                  onSelect={() => {
                    onSelect(repo.fullName === value ? null : repo);
                    setOpen(false);
                  }}
                  className="flex items-center gap-2"
                  aria-selected={value === repo.fullName}
                >
                  <RepoItem repo={repo} isSelected={value === repo.fullName} />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
