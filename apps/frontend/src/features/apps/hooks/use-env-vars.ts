import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { CreateEnvVarInput } from "@/types";

export function useEnvVars(appId: string) {
  return useQuery({
    queryKey: ["apps", appId, "env-vars"],
    queryFn: () => api.envVars.list(appId),
    enabled: !!appId,
  });
}

export function useCreateEnvVar(appId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateEnvVarInput) => api.envVars.create(appId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps", appId, "env-vars"] });
    },
  });
}

export function useBulkUpsertEnvVars(appId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (vars: readonly CreateEnvVarInput[]) =>
      api.envVars.bulkUpsert(appId, { vars }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps", appId, "env-vars"] });
    },
  });
}

export function useUpdateEnvVar(appId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      varId,
      input,
    }: {
      varId: string;
      input: Partial<CreateEnvVarInput>;
    }) => api.envVars.update(appId, varId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps", appId, "env-vars"] });
    },
  });
}

export function useDeleteEnvVar(appId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (varId: string) => api.envVars.delete(appId, varId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps", appId, "env-vars"] });
    },
  });
}
