import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronsUpDown, GitBranch, Lock } from "lucide-react";
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
import { cn } from "@/lib/utils";
import { type GitHubRepository, api } from "@/services/api";

interface RepoSelectorProps {
  readonly value?: string;
  readonly onSelect: (repo: GitHubRepository | null) => void;
  readonly installationId?: string;
}

export function RepoSelector({
  value,
  onSelect,
  installationId,
}: RepoSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["github", "repos", installationId],
    queryFn: () => api.github.repos(installationId),
    refetchOnWindowFocus: true,
    staleTime: 30 * 1000,
  });

  const filteredRepos = useMemo(() => {
    if (!data?.repositories) return [];
    if (!searchQuery) return data.repositories;

    const query = searchQuery.toLowerCase();
    return data.repositories.filter(
      (repo) =>
        repo.name.toLowerCase().includes(query) ||
        repo.fullName.toLowerCase().includes(query) ||
        repo.description?.toLowerCase().includes(query),
    );
  }, [data?.repositories, searchQuery]);

  const selectedRepo = data?.repositories.find(
    (repo) => repo.fullName === value,
  );

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-4 w-48" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-destructive">
        Failed to load repositories. Please try again.
      </div>
    );
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
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[400px] p-0" align="start">
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Search repositories..."
            value={searchQuery}
            onValueChange={setSearchQuery}
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
                >
                  <Check
                    className={cn(
                      "h-4 w-4",
                      value === repo.fullName ? "opacity-100" : "opacity-0",
                    )}
                  />
                  <div className="flex flex-col flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      {repo.private && (
                        <Lock className="h-3 w-3 text-muted-foreground shrink-0" />
                      )}
                      <span className="font-medium truncate">
                        {repo.fullName}
                      </span>
                    </div>
                    {repo.description && (
                      <span className="text-xs text-muted-foreground truncate">
                        {repo.description}
                      </span>
                    )}
                    <div className="flex items-center gap-3 text-xs text-muted-foreground mt-1">
                      {repo.language && <span>{repo.language}</span>}
                      <span className="flex items-center gap-1">
                        <GitBranch className="h-3 w-3" />
                        {repo.defaultBranch}
                      </span>
                    </div>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
