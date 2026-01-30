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
import type { App } from "@/types";

interface AppSettingsDialogProps {
  readonly app: App;
}

export function AppSettingsDialog({ app }: AppSettingsDialogProps) {
  const [open, setOpen] = useState(false);
  const [branch, setBranch] = useState(app.branch);
  const [workdir, setWorkdir] = useState(app.workdir);
  const updateApp = useUpdateApp();

  useEffect(() => {
    if (open) {
      setBranch(app.branch);
      setWorkdir(app.workdir);
    }
  }, [open, app.branch, app.workdir]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateApp.mutate(
      {
        id: app.id,
        input: {
          branch: branch !== app.branch ? branch : undefined,
          workdir: workdir !== app.workdir ? workdir : undefined,
        },
      },
      {
        onSuccess: () => setOpen(false),
      },
    );
  };

  const hasChanges = branch !== app.branch || workdir !== app.workdir;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Settings className="h-4 w-4 mr-2" />
          Settings
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
