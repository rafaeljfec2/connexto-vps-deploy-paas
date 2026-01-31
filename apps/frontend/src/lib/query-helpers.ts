import {
  type QueryKey,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

export const REFETCH_INTERVALS = {
  FAST: 5000,
  NORMAL: 10000,
  SLOW: 30000,
} as const;

export const STALE_TIMES = {
  SHORT: 30 * 1000,
  NORMAL: 60 * 1000,
  LONG: 5 * 60 * 1000,
} as const;

export const QUERY_KEYS = {
  apps: () => ["apps"] as const,
  app: (id: string) => ["app", id] as const,
  appHealth: (id: string) => ["app-health", id] as const,
  appUrl: (id: string) => ["appUrl", id] as const,
  appConfig: (id: string) => ["appConfig", id] as const,
  webhookStatus: (id: string) => ["webhookStatus", id] as const,
  deployments: (appId: string) => ["deployments", appId] as const,
  containerLogs: (appId: string, tail: number) =>
    ["containerLogs", appId, tail] as const,
  containerStats: (appId: string) => ["containerStats", appId] as const,
  commits: (appId: string, limit: number) => ["commits", appId, limit] as const,
  envVars: (appId: string) => ["envVars", appId] as const,
  github: {
    repos: (installationId?: string) =>
      ["github", "repos", installationId] as const,
    installations: () => ["github", "installations"] as const,
  },
} as const;

type InvalidateKeysFactory<TVariables> = (variables: TVariables) => QueryKey[];

export function useInvalidatingMutation<TData, TVariables>(
  mutationFn: (variables: TVariables) => Promise<TData>,
  getInvalidateKeys: InvalidateKeysFactory<TVariables>,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
    onSuccess: (_data, variables) => {
      const keys = getInvalidateKeys(variables);
      keys.forEach((key) => {
        queryClient.invalidateQueries({ queryKey: key });
      });
    },
  });
}

export function createAppInvalidateKeys(appId: string): QueryKey[] {
  return [QUERY_KEYS.apps(), QUERY_KEYS.app(appId)];
}

export function createAppWithWebhookInvalidateKeys(appId: string): QueryKey[] {
  return [
    QUERY_KEYS.apps(),
    QUERY_KEYS.app(appId),
    QUERY_KEYS.webhookStatus(appId),
  ];
}

export function createContainerInvalidateKeys(appId: string): QueryKey[] {
  return [QUERY_KEYS.appHealth(appId)];
}

export function createEnvVarInvalidateKeys(appId: string): QueryKey[] {
  return [QUERY_KEYS.envVars(appId)];
}
