export const ROUTES = {
  HOME: "/",
  LOGIN: "/login",
  NEW_APP: "/apps/new",
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
