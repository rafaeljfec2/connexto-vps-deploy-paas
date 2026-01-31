import { useState } from "react";
import { FolderGit2, GitBranch, Link } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FormField } from "@/components/form-field";
import type { GitHubRepository } from "@/services/api";
import { RepoSelector } from "../repo-selector";
import type { StepProps } from "./types";

export function RepositoryStep({
  data,
  onUpdate,
  onNext,
}: Readonly<StepProps>) {
  const [mode, setMode] = useState<"github" | "manual">("github");
  const [selectedRepo, setSelectedRepo] = useState<string | undefined>(
    undefined,
  );

  const isValid =
    data.name.length >= 2 &&
    (data.repositoryUrl.includes("github.com") ||
      data.repositoryUrl.includes("git@github.com"));

  const handleNameChange = (value: string) => {
    onUpdate({
      name: value.toLowerCase().replaceAll(/[^a-z0-9-]/g, "-"),
    });
  };

  const handleRepoSelect = (repo: GitHubRepository | null) => {
    if (repo) {
      setSelectedRepo(repo.fullName);
      const appName = repo.name.toLowerCase().replaceAll(/[^a-z0-9-]/g, "-");
      onUpdate({
        repositoryUrl: repo.cloneUrl,
        name: appName,
        branch: repo.defaultBranch,
      });
    } else {
      setSelectedRepo(undefined);
      onUpdate({
        repositoryUrl: "",
        name: "",
        branch: "main",
      });
    }
  };

  return (
    <Card className="border-0 shadow-none md:border md:shadow-sm">
      <CardContent className="p-0 md:p-6 space-y-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <FolderGit2 className="h-5 w-5" />
            <h3 className="font-semibold">Connect your repository</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Select a GitHub repository or enter the URL manually.
          </p>
        </div>

        <Tabs
          value={mode}
          onValueChange={(v) => setMode(v as "github" | "manual")}
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="github">From GitHub</TabsTrigger>
            <TabsTrigger value="manual">Manual URL</TabsTrigger>
          </TabsList>

          <TabsContent value="github" className="space-y-4 mt-4">
            <FormField label="Select Repository" htmlFor="repo-select" required>
              <RepoSelector value={selectedRepo} onSelect={handleRepoSelect} />
            </FormField>
          </TabsContent>

          <TabsContent value="manual" className="space-y-4 mt-4">
            <FormField label="Repository URL" htmlFor="repository" required>
              <div className="relative">
                <Link className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="repository"
                  placeholder="https://github.com/owner/repo"
                  value={data.repositoryUrl}
                  onChange={(e) => onUpdate({ repositoryUrl: e.target.value })}
                  className="pl-10"
                  required
                />
              </div>
            </FormField>
          </TabsContent>
        </Tabs>

        <FormField
          label="Application Name"
          htmlFor="name"
          helper="This will be used as the subdomain for your app."
          required
        >
          <Input
            id="name"
            placeholder="my-awesome-app"
            value={data.name}
            onChange={(e) => handleNameChange(e.target.value)}
            required
            minLength={2}
            maxLength={63}
          />
        </FormField>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormField
            label="Branch"
            htmlFor="branch"
            helper="The branch to deploy from."
          >
            <div className="relative">
              <GitBranch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                id="branch"
                placeholder="main"
                value={data.branch}
                onChange={(e) => onUpdate({ branch: e.target.value })}
                className="pl-10"
              />
            </div>
          </FormField>

          <FormField
            label="Working Directory"
            htmlFor="workdir"
            helper="For monorepos, e.g. apps/api"
          >
            <div className="relative">
              <FolderGit2 className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                id="workdir"
                placeholder="(root)"
                value={data.workdir}
                onChange={(e) => onUpdate({ workdir: e.target.value })}
                className="pl-10"
              />
            </div>
          </FormField>
        </div>
      </CardContent>

      <CardFooter className="p-0 pt-6 md:p-6 md:pt-0">
        <Button
          className="w-full md:w-auto md:ml-auto"
          onClick={onNext}
          disabled={!isValid}
        >
          Continue to Environment
        </Button>
      </CardFooter>
    </Card>
  );
}
