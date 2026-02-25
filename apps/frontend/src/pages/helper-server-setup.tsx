import { ROUTES } from "@/constants/routes";
import { AlertCircle, Server } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { PageHeader } from "@/components/page-header";
import {
  AddServerSection,
  ConfigureBackendSection,
  FilesStructureSection,
  FirewallPortsSection,
  PrepareRemoteSection,
  ProvisionVerifySection,
  ProvisionWorkflowSection,
  QuickChecklistSection,
  TroubleshootingSection,
} from "@/features/servers/components/helper-server-setup";

export function HelperServerSetupPage() {
  return (
    <div className="space-y-8">
      <PageHeader
        backTo={ROUTES.SERVERS}
        title="Server setup guide"
        description="Step-by-step tutorial to prepare a remote server, provision Docker, Traefik, and the deploy agent"
        icon={Server}
      />

      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Overview</AlertTitle>
        <AlertDescription>
          Prepare the remote VPS, configure the backend, add the server in the
          panel, then provision. The provisioning{" "}
          <strong>automatically installs Docker, Traefik, and the agent</strong>{" "}
          via SSH. The agent connects to the backend via gRPC with mTLS.
        </AlertDescription>
      </Alert>

      <ProvisionWorkflowSection />
      <FirewallPortsSection />
      <PrepareRemoteSection />
      <ConfigureBackendSection />
      <AddServerSection />
      <ProvisionVerifySection />
      <TroubleshootingSection />
      <QuickChecklistSection />
      <FilesStructureSection />
    </div>
  );
}
