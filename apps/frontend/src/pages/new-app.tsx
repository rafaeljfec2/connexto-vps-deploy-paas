import { useSearchParams } from "react-router-dom";
import { OnboardingWizard } from "@/features/apps/components/onboarding";

export function NewAppPage() {
  const [searchParams] = useSearchParams();
  const initialServerId = searchParams.get("serverId") ?? undefined;

  return <OnboardingWizard initialServerId={initialServerId} />;
}
