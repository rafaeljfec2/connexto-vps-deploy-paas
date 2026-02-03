import { useState } from "react";
import { Plus } from "lucide-react";
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
import { useCreateServer } from "../hooks/use-servers";

interface AddServerDialogProps {
  readonly trigger?: React.ReactNode;
}

export function AddServerDialog({ trigger }: AddServerDialogProps) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [host, setHost] = useState("");
  const [sshPort, setSshPort] = useState("22");
  const [sshUser, setSshUser] = useState("");
  const [sshKey, setSshKey] = useState("");

  const createServer = useCreateServer();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createServer.mutateAsync({
        name,
        host,
        sshPort: Number.parseInt(sshPort, 10) || 22,
        sshUser,
        sshKey,
      });
      setOpen(false);
      setName("");
      setHost("");
      setSshPort("22");
      setSshUser("");
      setSshKey("");
    } catch {
      // Error handled by mutation
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button size="sm">
            <Plus className="h-4 w-4 mr-2" aria-hidden />
            Add Server
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Remote Server</DialogTitle>
            <DialogDescription>
              Add a server for remote deploy. You will need SSH access with key
              authentication.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label
                htmlFor="name"
                className="text-sm font-medium leading-none"
              >
                Name
              </label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="production"
                required
              />
            </div>
            <div className="grid gap-2">
              <label
                htmlFor="host"
                className="text-sm font-medium leading-none"
              >
                Host
              </label>
              <Input
                id="host"
                value={host}
                onChange={(e) => setHost(e.target.value)}
                placeholder="192.168.1.10 or server.example.com"
                required
              />
            </div>
            <div className="grid gap-2">
              <label
                htmlFor="sshPort"
                className="text-sm font-medium leading-none"
              >
                SSH Port
              </label>
              <Input
                id="sshPort"
                type="number"
                value={sshPort}
                onChange={(e) => setSshPort(e.target.value)}
                placeholder="22"
                min={1}
                max={65535}
              />
            </div>
            <div className="grid gap-2">
              <label
                htmlFor="sshUser"
                className="text-sm font-medium leading-none"
              >
                SSH User
              </label>
              <Input
                id="sshUser"
                value={sshUser}
                onChange={(e) => setSshUser(e.target.value)}
                placeholder="root"
                required
              />
            </div>
            <div className="grid gap-2">
              <label
                htmlFor="sshKey"
                className="text-sm font-medium leading-none"
              >
                SSH Private Key
              </label>
              <textarea
                id="sshKey"
                value={sshKey}
                onChange={(e) => setSshKey(e.target.value)}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                rows={4}
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                required
              />
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
            <Button
              type="submit"
              disabled={
                createServer.isPending || !name || !host || !sshUser || !sshKey
              }
            >
              {createServer.isPending ? "Adding..." : "Add Server"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
