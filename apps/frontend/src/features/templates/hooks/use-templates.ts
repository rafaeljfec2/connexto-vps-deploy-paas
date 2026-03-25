import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/query-config";
import { api } from "@/services/api";
import type { DeployTemplateInput } from "@/types";

export function useTemplates(category?: string) {
  return useQuery({
    queryKey: ["templates", category],
    queryFn: () => api.templates.list(category),
    staleTime: STALE_TIMES.LONG,
  });
}

export function useTemplate(id: string | undefined) {
  return useQuery({
    queryKey: ["templates", id],
    queryFn: () => api.templates.get(id!),
    enabled: Boolean(id),
    staleTime: STALE_TIMES.LONG,
  });
}

export function useDeployTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      input,
      serverId,
    }: {
      readonly id: string;
      readonly input: DeployTemplateInput;
      readonly serverId?: string;
    }) => api.templates.deploy(id, input, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
