import { useCallback, useEffect, useState } from "react";

export type SectionKey =
  | "deployments"
  | "containerLogs"
  | "metrics"
  | "envVars"
  | "health"
  | "config"
  | "webhook"
  | "domains"
  | "networks"
  | "volumes";

const STORAGE_KEY = "app-details-sections";

const DEFAULT_STATE: Record<SectionKey, boolean> = {
  deployments: true,
  containerLogs: true,
  metrics: false,
  envVars: false,
  health: false,
  config: false,
  webhook: false,
  domains: false,
  networks: false,
  volumes: false,
};

function loadPersistedState(): Record<SectionKey, boolean> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Record<string, boolean>;
      return { ...DEFAULT_STATE, ...parsed };
    }
  } catch {
    // ignore corrupted storage
  }
  return DEFAULT_STATE;
}

export function useExpandedSections() {
  const [expandedSections, setExpandedSections] =
    useState<Record<SectionKey, boolean>>(loadPersistedState);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(expandedSections));
  }, [expandedSections]);

  const toggleSection = useCallback((section: SectionKey) => {
    setExpandedSections((prev) => ({ ...prev, [section]: !prev[section] }));
  }, []);

  const allExpanded = Object.values(expandedSections).every(Boolean);

  const toggleAllSections = useCallback(() => {
    setExpandedSections((prev) => {
      const allOn = Object.values(prev).every(Boolean);
      const newState = !allOn;
      return Object.fromEntries(
        Object.keys(prev).map((k) => [k, newState]),
      ) as Record<SectionKey, boolean>;
    });
  }, []);

  return { expandedSections, toggleSection, allExpanded, toggleAllSections };
}
