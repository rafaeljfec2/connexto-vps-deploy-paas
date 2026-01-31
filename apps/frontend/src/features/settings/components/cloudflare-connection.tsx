import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Cloud, ExternalLink, Loader2, Unlink } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api } from "@/services/api";

export function CloudflareConnection() {
  const queryClient = useQueryClient();
  const [apiToken, setApiToken] = useState("");
  const [showTokenInput, setShowTokenInput] = useState(false);

  const { data: status, isLoading } = useQuery({
    queryKey: ["cloudflare-status"],
    queryFn: () => api.cloudflare.status(),
  });

  const connectMutation = useMutation({
    mutationFn: (token: string) => api.cloudflare.connect(token),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cloudflare-status"] });
      setApiToken("");
      setShowTokenInput(false);
    },
  });

  const disconnectMutation = useMutation({
    mutationFn: () => api.cloudflare.disconnect(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cloudflare-status"] });
    },
  });

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    if (apiToken.trim()) {
      connectMutation.mutate(apiToken.trim());
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Cloud className="h-5 w-5 text-orange-500" />
          Cloudflare
        </CardTitle>
        <CardDescription>
          Connect your Cloudflare account to automatically configure DNS for
          custom domains
        </CardDescription>
      </CardHeader>
      <CardContent>
        {status?.connected ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900">
                  <Check className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
                <div>
                  <p className="font-medium">Connected</p>
                  <p className="text-sm text-muted-foreground">
                    {status.email}
                  </p>
                </div>
              </div>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={disconnectMutation.isPending}
                  >
                    {disconnectMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Unlink className="h-4 w-4 mr-2" />
                    )}
                    Disconnect
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Disconnect Cloudflare</AlertDialogTitle>
                    <AlertDialogDescription>
                      Are you sure you want to disconnect your Cloudflare
                      account? You will no longer be able to manage DNS records
                      for custom domains automatically.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={() => disconnectMutation.mutate()}
                      className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                      Disconnect
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        ) : showTokenInput ? (
          <form onSubmit={handleConnect} className="space-y-4">
            <div className="space-y-2">
              <label
                htmlFor="apiToken"
                className="text-sm font-medium leading-none"
              >
                Cloudflare API Token
              </label>
              <Input
                id="apiToken"
                type="password"
                placeholder="Enter your Cloudflare API token"
                value={apiToken}
                onChange={(e) => setApiToken(e.target.value)}
                disabled={connectMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                Create an API token with <strong>Zone.DNS Edit</strong>{" "}
                permission.{" "}
                <a
                  href="https://dash.cloudflare.com/profile/api-tokens"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline inline-flex items-center gap-1"
                >
                  Create token
                  <ExternalLink className="h-3 w-3" />
                </a>
              </p>
            </div>

            {connectMutation.isError && (
              <p className="text-sm text-destructive">
                {connectMutation.error instanceof Error
                  ? connectMutation.error.message
                  : "Failed to connect"}
              </p>
            )}

            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowTokenInput(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={!apiToken.trim() || connectMutation.isPending}
              >
                {connectMutation.isPending && (
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                )}
                Connect
              </Button>
            </div>
          </form>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Connect your Cloudflare account to enable automatic DNS
              configuration when adding custom domains to your applications.
            </p>
            <Button onClick={() => setShowTokenInput(true)}>
              <Cloud className="h-4 w-4 mr-2" />
              Connect Cloudflare
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
