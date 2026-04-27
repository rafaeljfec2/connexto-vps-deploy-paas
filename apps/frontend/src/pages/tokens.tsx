import { KeyRound } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { TokenList } from "@/features/tokens";

export function TokensPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Personal access tokens"
        description="Create tokens to authenticate API clients, MCP servers and CI pipelines with scoped permissions."
        icon={KeyRound}
        backTo="/settings"
      />
      <TokenList />
    </div>
  );
}
