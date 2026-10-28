import type { App, Deployment } from "@/types";

/**
 * Production version shown in the app hero: prefer the first successful
 * deployment's appVersion (same "current" rule as DeployList), then fall
 * back to the app-level field persisted on last success.
 */
export function resolveProductionAppVersion(
  app: Pick<App, "appVersion">,
  deployments: readonly Deployment[] | null | undefined,
): string | null {
  const currentSuccess = deployments?.find(
    (deployment) => deployment.status === "success",
  );
  const fromDeploy = currentSuccess?.appVersion?.trim();
  if (fromDeploy !== undefined && fromDeploy !== "") {
    return fromDeploy;
  }

  const fromApp = app.appVersion?.trim();
  if (fromApp !== undefined && fromApp !== "") {
    return fromApp;
  }

  return null;
}
