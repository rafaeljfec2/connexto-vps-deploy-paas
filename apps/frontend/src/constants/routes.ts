export const ROUTES = {
  HOME: "/",
  LOGIN: "/login",
  NEW_APP: "/apps/new",
  SETTINGS: "/settings",
  APP_DETAIL: (id: string) => `/apps/${id}`,
} as const;

export const API_ROUTES = {
  AUTH: {
    GITHUB: "/auth/github",
    ME: "/auth/me",
    LOGOUT: "/auth/logout",
  },
  GITHUB: {
    INSTALL: "/api/github/install",
    REPOS: "/api/github/repos",
    INSTALLATIONS: "/api/github/installations",
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
