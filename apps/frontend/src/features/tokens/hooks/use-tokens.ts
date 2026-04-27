import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  type CreateTokenPayload,
  type CreateTokenResponse,
  type ListTokensResponse,
  createToken,
  listTokens,
  revokeToken,
} from "@/services/api/tokens";

const TOKENS_QUERY_KEY = ["personal-access-tokens"] as const;

export function usePersonalAccessTokens() {
  return useQuery<ListTokensResponse, Error>({
    queryKey: TOKENS_QUERY_KEY,
    queryFn: listTokens,
  });
}

export function useCreateToken() {
  const queryClient = useQueryClient();
  return useMutation<CreateTokenResponse, Error, CreateTokenPayload>({
    mutationFn: createToken,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: TOKENS_QUERY_KEY });
    },
  });
}

export function useRevokeToken() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: revokeToken,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: TOKENS_QUERY_KEY });
    },
  });
}
