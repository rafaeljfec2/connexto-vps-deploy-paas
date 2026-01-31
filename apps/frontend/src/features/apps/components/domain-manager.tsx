import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Cloud,
  ExternalLink,
  Globe,
  Loader2,
  Plus,
  Trash2,
} from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
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
import type { CustomDomain } from "@/types";

interface DomainManagerProps {
  readonly appId: string;
}

export function DomainManager({ appId }: DomainManagerProps) {
  const queryClient = useQueryClient();
  const [newDomain, setNewDomain] = useState("");
  const [domainToDelete, setDomainToDelete] = useState<CustomDomain | null>(
    null,
  );

  const { data: cloudflareStatus, isLoading: isLoadingCloudflare } = useQuery({
    queryKey: ["cloudflare-status"],
    queryFn: () => api.cloudflare.status(),
  });

  const { data: domains = [], isLoading: isLoadingDomains } = useQuery({
    queryKey: ["custom-domains", appId],
    queryFn: () => api.domains.list(appId),
  });

  const addDomainMutation = useMutation({
    mutationFn: (domain: string) => api.domains.add(appId, domain),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom-domains", appId] });
      setNewDomain("");
    },
  });

  const removeDomainMutation = useMutation({
    mutationFn: (domainId: string) => api.domains.remove(appId, domainId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom-domains", appId] });
      setDomainToDelete(null);
    },
  });

  const handleAddDomain = (e: React.FormEvent) => {
    e.preventDefault();
    if (newDomain.trim()) {
      addDomainMutation.mutate(newDomain.trim());
    }
  };

  const isLoading = isLoadingCloudflare || isLoadingDomains;

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (!cloudflareStatus?.connected) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Custom Domains
          </CardTitle>
          <CardDescription>
            Add custom domains to your application
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center gap-4 py-8 text-center">
            <Cloud className="h-12 w-12 text-muted-foreground" />
            <div>
              <p className="font-medium">Cloudflare Not Connected</p>
              <p className="text-sm text-muted-foreground">
                Connect your Cloudflare account in Settings to manage custom
                domains
              </p>
            </div>
            <Button variant="outline" asChild>
              <a href="/settings">Go to Settings</a>
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Custom Domains
          </CardTitle>
          <CardDescription>
            Add custom domains from your Cloudflare account
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleAddDomain} className="flex gap-2">
            <Input
              placeholder="example.com or sub.example.com"
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              disabled={addDomainMutation.isPending}
            />
            <Button
              type="submit"
              disabled={!newDomain.trim() || addDomainMutation.isPending}
            >
              {addDomainMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
              <span className="ml-2 hidden sm:inline">Add</span>
            </Button>
          </form>

          {addDomainMutation.isError && (
            <p className="text-sm text-destructive">
              {addDomainMutation.error instanceof Error
                ? addDomainMutation.error.message
                : "Failed to add domain"}
            </p>
          )}

          {domains.length > 0 ? (
            <div className="space-y-2">
              {domains.map((domain) => (
                <div
                  key={domain.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <Globe className="h-4 w-4 text-muted-foreground" />
                    <div>
                      <a
                        href={`https://${domain.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-medium hover:underline flex items-center gap-1"
                      >
                        {domain.domain}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Badge variant="secondary" className="text-xs">
                          {domain.recordType}
                        </Badge>
                        <span>Proxied via Cloudflare</span>
                      </div>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setDomainToDelete(domain)}
                    disabled={removeDomainMutation.isPending}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              ))}
            </div>
          ) : (
            <div className="py-8 text-center text-sm text-muted-foreground">
              No custom domains configured yet
            </div>
          )}
        </CardContent>
      </Card>

      <AlertDialog
        open={!!domainToDelete}
        onOpenChange={() => setDomainToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Domain</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove{" "}
              <strong>{domainToDelete?.domain}</strong>? This will also delete
              the DNS record from Cloudflare.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                domainToDelete && removeDomainMutation.mutate(domainToDelete.id)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeDomainMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : null}
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
