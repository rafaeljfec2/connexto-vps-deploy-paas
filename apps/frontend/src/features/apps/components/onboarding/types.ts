import { ClipboardCheck, FolderGit2, Rocket, Variable } from "lucide-react";
import type { CreateEnvVarInput } from "@/types";

export interface OnboardingData {
  readonly name: string;
  readonly repositoryUrl: string;
  readonly branch: string;
  readonly workdir: string;
  readonly envVars: readonly LocalEnvVar[];
}

export interface LocalEnvVar extends CreateEnvVarInput {
  readonly localId: string;
}

export interface StepProps {
  readonly data: OnboardingData;
  readonly onUpdate: (updates: Partial<OnboardingData>) => void;
  readonly onNext: () => void;
  readonly onBack: () => void;
}

export const ONBOARDING_STEPS = [
  { id: "repository", title: "Repository", icon: FolderGit2 },
  { id: "environment", title: "Environment", icon: Variable },
  { id: "review", title: "Review", icon: ClipboardCheck },
  { id: "deploy", title: "Deploy", icon: Rocket },
] as const;

export const INITIAL_DATA: OnboardingData = {
  name: "",
  repositoryUrl: "",
  branch: "main",
  workdir: "",
  envVars: [],
};
