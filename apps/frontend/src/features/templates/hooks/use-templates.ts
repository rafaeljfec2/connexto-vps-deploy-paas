import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { DeployTemplateInput } from "@/types";

export function useTemplates(category?: string) {
  return useQuery({
    queryKey: ["templates", category],
    queryFn: () => api.templates.list(category),
    staleTime: 5 * 60 * 1000,
  });
}

export function useTemplate(id: string | undefined) {
  return useQuery({
    queryKey: ["templates", id],
    queryFn: () => api.templates.get(id!),
    enabled: Boolean(id),
    staleTime: 5 * 60 * 1000,
  });
}

export function useDeployTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: DeployTemplateInput }) =>
      api.templates.deploy(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
