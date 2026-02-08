import { useEffect, useReducer } from "react";
import type { AgentUpdateState } from "@/types";
import {
  getAgentUpdateState,
  subscribeAgentUpdate,
} from "../agent-update-store";

export function useAgentUpdate(
  serverId: string | undefined,
): AgentUpdateState | undefined {
  const [, forceRender] = useReducer((x: number) => x + 1, 0);

  useEffect(() => {
    if (!serverId) return;
    return subscribeAgentUpdate(forceRender);
  }, [serverId, forceRender]);

  return serverId ? getAgentUpdateState(serverId) : undefined;
}
