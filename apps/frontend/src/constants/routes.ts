export const ROUTES = {
  HOME: "/",
  LOGIN: "/login",
  NEW_APP: "/apps/new",
  SETTINGS: "/settings",
  APP_DETAIL: (id: string) => `/apps/${id}`,
} as const;

const API_URL = import.meta.env.VITE_API_URL ?? "";

export const API_ROUTES = {
  AUTH: {
    GITHUB: `${API_URL}/auth/github`,
    ME: `${API_URL}/auth/me`,
    LOGOUT: `${API_URL}/auth/logout`,
  },
  GITHUB: {
    INSTALL: `${API_URL}/api/github/install`,
    REPOS: `${API_URL}/api/github/repos`,
    INSTALLATIONS: `${API_URL}/api/github/installations`,
  },
} as const;

export const STALE_TIMES = {
  SHORT: 30 * 1000,
  NORMAL: 60 * 1000,
  LONG: 5 * 60 * 1000,
} as const;

export const DEFAULTS = {
  COMMITS_LIMIT: 20 as number,
  LOGS_TAIL: 100 as number,
};
