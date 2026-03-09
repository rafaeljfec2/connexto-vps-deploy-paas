import { useEffect, useState } from "react";
import { Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useUpdateApp } from "@/features/apps/hooks/use-apps";

interface AppSettingsDialogProps {
  readonly app: {
    readonly id: string;
    readonly name: string;
    readonly branch: string;
    readonly workdir: string;
  };
}

export function AppSettingsDialog({ app }: AppSettingsDialogProps) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState(app.name);
  const [branch, setBranch] = useState(app.branch);
  const [workdir, setWorkdir] = useState(app.workdir);
  const updateApp = useUpdateApp();

  useEffect(() => {
    if (open) {
      setName(app.name);
      setBranch(app.branch);
      setWorkdir(app.workdir);
    }
  }, [open, app.name, app.branch, app.workdir]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const nameChanged = name.trim() !== app.name;
    const branchChanged = branch !== app.branch;
    const workdirChanged = workdir !== app.workdir;
    updateApp.mutate(
      {
        id: app.id,
        input: {
          name: nameChanged ? name.trim() : undefined,
          branch: branchChanged ? branch : undefined,
          workdir: workdirChanged ? workdir : undefined,
        },
      },
      {
        onSuccess: () => setOpen(false),
      },
    );
  };

  const hasChanges =
    name.trim() !== app.name ||
    branch !== app.branch ||
    workdir !== app.workdir;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Settings className="h-4 w-4" />
          <span className="hidden lg:inline ml-2">Settings</span>
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>App Settings</DialogTitle>
          <DialogDescription>
            Edit the configuration for {app.name}. Changes will take effect on
            the next deploy.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label htmlFor="app-name" className="text-sm font-medium">
                Application Name
              </label>
              <Input
                id="app-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-app"
              />
              <p className="text-xs text-muted-foreground">
                The display name for this application
              </p>
            </div>
            <div className="space-y-2">
              <label htmlFor="branch" className="text-sm font-medium">
                Branch
              </label>
              <Input
                id="branch"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder="main"
              />
              <p className="text-xs text-muted-foreground">
                The Git branch to deploy from
              </p>
            </div>
            <div className="space-y-2">
              <label htmlFor="workdir" className="text-sm font-medium">
                Working Directory
              </label>
              <Input
                id="workdir"
                value={workdir}
                onChange={(e) => setWorkdir(e.target.value)}
                placeholder="."
                className="font-mono"
              />
              <p className="text-xs text-muted-foreground">
                Path to the app directory (relative to repository root)
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!hasChanges || updateApp.isPending}>
              {updateApp.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
