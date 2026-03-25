import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Check, Copy, Loader2, Lock, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/services/api";
import type { SSLStatusResult } from "@/services/api/infrastructure";

const DB_IMAGES = [
  "postgres",
  "postgresql",
  "mysql",
  "mariadb",
  "mongo",
] as const;

function detectDatabaseType(image: string): string | undefined {
  const lower = image.toLowerCase();
  if (lower.includes("postgres")) return "postgresql";
  if (lower.includes("mysql") || lower.includes("mariadb")) return "mysql";
  if (lower.includes("mongo")) return "mongodb";
  return undefined;
}

export function isDatabaseImage(image: string): boolean {
  const lower = image.toLowerCase();
  return DB_IMAGES.some((db) => lower.includes(db));
}

interface ContainerSSLDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly containerId: string;
  readonly containerName: string;
  readonly containerImage: string;
  readonly serverId: string;
  readonly serverHost: string;
  readonly containerPort?: number;
}

export function ContainerSSLDialog({
  open,
  onOpenChange,
  containerId,
  containerName,
  containerImage,
  serverId,
  serverHost,
  containerPort,
}: ContainerSSLDialogProps) {
  const dbType = detectDatabaseType(containerImage) ?? "postgresql";

  const [dbUser, setDbUser] = useState("");
  const [dbName, setDbName] = useState("");
  const [dbPassword, setDbPassword] = useState("");
  const [copied, setCopied] = useState(false);
  const [result, setResult] = useState<SSLStatusResult | null>(null);

  const configureSSL = useMutation({
    mutationFn: () =>
      api.containerSSL.configure(containerId, {
        serverId,
        databaseType: dbType,
        databaseUser: dbUser,
        databaseName: dbName,
      }),
    onSuccess: (data) => {
      const port = containerPort ?? 5432;
      const connStr = `postgresql://${dbUser}:${dbPassword}@${serverHost}:${port}/${dbName}?sslmode=require`;
      setResult({ ...data, connectionString: connStr });
    },
  });

  const handleCopy = () => {
    if (result?.connectionString) {
      navigator.clipboard.writeText(result.connectionString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClose = (value: boolean) => {
    if (!value) {
      setResult(null);
      setDbUser("");
      setDbName("");
      setDbPassword("");
      setCopied(false);
    }
    onOpenChange(value);
  };

  const isValid = dbUser.trim() !== "" && dbName.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-emerald-500" />
            Configure SSL — {containerName}
          </DialogTitle>
          <DialogDescription>
            Enable encrypted connections for external access. After
            configuration, only SSL connections will be accepted from outside.
          </DialogDescription>
        </DialogHeader>

        {result ? (
          <div className="space-y-4 py-2">
            <div className="rounded-md bg-emerald-500/10 border border-emerald-500/20 p-4">
              <div className="flex items-center gap-2 mb-3">
                <Lock className="h-4 w-4 text-emerald-500" />
                <span className="font-medium text-emerald-500">
                  SSL Enabled
                </span>
                {result.tlsVersion && (
                  <span className="text-xs bg-emerald-500/20 text-emerald-400 rounded px-1.5 py-0.5">
                    {result.tlsVersion}
                  </span>
                )}
              </div>

              <Label className="text-xs text-muted-foreground mb-1 block">
                Connection String
              </Label>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-background rounded p-2 font-mono break-all select-all border">
                  {result.connectionString}
                </code>
                <Button
                  variant="outline"
                  size="icon"
                  className="shrink-0 h-8 w-8"
                  onClick={handleCopy}
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-emerald-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {result.certificateExpiry && (
              <p className="text-xs text-muted-foreground">
                Certificate expires: {result.certificateExpiry}
              </p>
            )}
          </div>
        ) : (
          <div className="space-y-4 py-2">
            <div className="rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
              <p>
                Database type detected:{" "}
                <span className="font-medium text-foreground">{dbType}</span>{" "}
                from image{" "}
                <code className="text-xs bg-muted rounded px-1">
                  {containerImage}
                </code>
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="ssl-db-user">Database User</Label>
              <Input
                id="ssl-db-user"
                placeholder="e.g. postgres"
                value={dbUser}
                onChange={(e) => setDbUser(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="ssl-db-name">Database Name</Label>
              <Input
                id="ssl-db-name"
                placeholder="e.g. mydb"
                value={dbName}
                onChange={(e) => setDbName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="ssl-db-password">
                Database Password{" "}
                <span className="text-muted-foreground text-xs">
                  (for connection string only, not sent to server)
                </span>
              </Label>
              <Input
                id="ssl-db-password"
                type="password"
                placeholder="Your database password"
                value={dbPassword}
                onChange={(e) => setDbPassword(e.target.value)}
                autoComplete="off"
              />
            </div>
          </div>
        )}

        <DialogFooter>
          {result ? (
            <Button variant="outline" onClick={() => handleClose(false)}>
              Close
            </Button>
          ) : (
            <Button
              onClick={() => configureSSL.mutate()}
              disabled={!isValid || configureSSL.isPending}
            >
              {configureSSL.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Configuring...
                </>
              ) : (
                <>
                  <ShieldCheck className="mr-2 h-4 w-4" />
                  Enable SSL
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
