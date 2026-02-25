import {
  useRemoveWebhook,
  useRestartContainer,
  useSetupWebhook,
  useStartContainer,
  useStopContainer,
} from "@/features/apps/hooks/use-apps";
import { useRedeploy, useRollback } from "@/features/deploys/hooks/use-deploys";

export function useAppActions(id: string | undefined) {
  const redeploy = useRedeploy();
  const rollback = useRollback();
  const setupWebhook = useSetupWebhook();
  const removeWebhook = useRemoveWebhook();
  const restartContainer = useRestartContainer();
  const stopContainer = useStopContainer();
  const startContainer = useStartContainer();

  const handleRedeploy = (sha?: string) => {
    if (id) redeploy.mutate({ appId: id, commitSha: sha });
  };

  const handleRollback = () => {
    if (id) rollback.mutate(id);
  };

  const handleSetupWebhook = () => {
    if (id) setupWebhook.mutate(id);
  };

  const handleRemoveWebhook = () => {
    if (id) removeWebhook.mutate(id);
  };

  return {
    redeploy,
    rollback,
    setupWebhook,
    removeWebhook,
    restartContainer,
    stopContainer,
    startContainer,
    handleRedeploy,
    handleRollback,
    handleSetupWebhook,
    handleRemoveWebhook,
  };
}
