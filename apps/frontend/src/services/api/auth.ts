import type { User } from "@/contexts/auth-context";
import { API_URL, fetchApi, fetchApiList } from "./client";

export interface RegisterInput {
  readonly email: string;
  readonly password: string;
  readonly name: string;
}

export interface LoginInput {
  readonly email: string;
  readonly password: string;
}

export interface GitHubInstallation {
  readonly id: string;
  readonly installationId: number;
  readonly accountType: string;
  readonly accountLogin: string;
  readonly repositorySelection: string;
}

export interface GitHubRepository {
  readonly id: number;
  readonly name: string;
  readonly fullName: string;
  readonly private: boolean;
  readonly description: string;
  readonly htmlUrl: string;
  readonly cloneUrl: string;
  readonly defaultBranch: string;
  readonly language: string;
  readonly owner: {
    readonly login: string;
    readonly avatarUrl: string;
    readonly type: string;
  };
}

export interface ReposResponse {
  readonly repositories: readonly GitHubRepository[];
  readonly needInstall: boolean;
  readonly installMessage?: string;
}

export const authApi = {
  me: (): Promise<User> => fetchApi<User>(`${API_URL}/auth/me`),

  register: (data: RegisterInput): Promise<User> =>
    fetchApi<User>(`${API_URL}/auth/register`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  login: (data: LoginInput): Promise<User> =>
    fetchApi<User>(`${API_URL}/auth/login`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  linkGitHub: (): Promise<{ redirectUrl: string }> =>
    fetchApi<{ redirectUrl: string }>(`${API_URL}/auth/link-github`, {
      method: "POST",
    }),

  logout: async (): Promise<void> => {
    await fetch(`${API_URL}/auth/logout`, {
      method: "POST",
      credentials: "include",
    });
  },
};

export const githubApi = {
  installations: (): Promise<readonly GitHubInstallation[]> =>
    fetchApiList<GitHubInstallation>(`${API_URL}/api/github/installations`),

  repos: (installationId?: string): Promise<ReposResponse> => {
    const url = installationId
      ? `${API_URL}/api/github/repos?installation_id=${installationId}`
      : `${API_URL}/api/github/repos`;
    return fetchApi<ReposResponse>(url);
  },

  repo: (owner: string, repo: string): Promise<GitHubRepository> =>
    fetchApi<GitHubRepository>(`${API_URL}/api/github/repos/${owner}/${repo}`),
};
