import { useEffect, useState } from "react";
import type { ProvisionProgressState } from "@/types";
import {
  getProvisionProgress,
  subscribeProvisionProgress,
} from "../provision-progress-store";

export function useProvisionProgress(
  serverId: string | undefined,
): ProvisionProgressState | undefined {
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!serverId) return;
    return subscribeProvisionProgress(() => setTick((n) => n + 1));
  }, [serverId]);

  return serverId ? getProvisionProgress(serverId) : undefined;
}
