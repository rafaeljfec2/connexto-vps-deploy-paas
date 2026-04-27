import { useMemo, useState } from "react";
import { KeyRound, Loader2, Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import type { PersonalAccessToken } from "@/services/api/tokens";
import { usePersonalAccessTokens } from "../hooks/use-tokens";
import { CreateTokenDialog } from "./create-token-dialog";
import { RevokeTokenButton } from "./revoke-token-button";

export function TokenList() {
  const [createOpen, setCreateOpen] = useState(false);
  const { data, isLoading, error } = usePersonalAccessTokens();

  const tokens = useMemo(() => data?.tokens ?? [], [data]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
      </div>
    );
  }

  if (error) {
    return <ErrorMessage message={error.message} />;
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New token
        </Button>
      </div>

      {tokens.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No personal access tokens"
          description="Create a token to authenticate CLI tools, MCP clients and CI pipelines with the FlowDeploy API."
          action={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create your first token
            </Button>
          }
        />
      ) : (
        <ul className="grid gap-3">
          {tokens.map((token) => (
            <li key={token.id}>
              <TokenCard token={token} />
            </li>
          ))}
        </ul>
      )}

      <CreateTokenDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}

interface TokenCardProps {
  readonly token: PersonalAccessToken;
}

function TokenCard({ token }: Readonly<TokenCardProps>) {
  const statusInfo = resolveStatus(token);

  return (
    <article className="bg-card flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="truncate font-semibold">{token.name}</h3>
          <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
        </div>
        <code className="text-muted-foreground block break-all font-mono text-xs">
          {token.tokenPrefix}…
        </code>
        <div className="flex flex-wrap gap-1">
          {token.scopes.map((scope) => (
            <Badge key={scope} variant="outline" className="text-xs">
              {scope}
            </Badge>
          ))}
        </div>
        <dl className="text-muted-foreground grid gap-1 text-xs sm:grid-cols-3">
          <div>
            <dt className="font-medium">Created</dt>
            <dd>{formatDate(token.createdAt)}</dd>
          </div>
          <div>
            <dt className="font-medium">Last used</dt>
            <dd>{token.lastUsedAt ? formatDate(token.lastUsedAt) : "Never"}</dd>
          </div>
          <div>
            <dt className="font-medium">Expires</dt>
            <dd>{token.expiresAt ? formatDate(token.expiresAt) : "Never"}</dd>
          </div>
        </dl>
      </div>

      {!token.revokedAt ? (
        <RevokeTokenButton tokenId={token.id} tokenName={token.name} />
      ) : null}
    </article>
  );
}

type StatusVariant = "default" | "secondary" | "destructive" | "outline";

function resolveStatus(token: PersonalAccessToken): {
  label: string;
  variant: StatusVariant;
} {
  if (token.revokedAt) {
    return { label: "Revoked", variant: "destructive" };
  }
  if (token.expiresAt && new Date(token.expiresAt).getTime() < Date.now()) {
    return { label: "Expired", variant: "secondary" };
  }
  return { label: "Active", variant: "default" };
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}
