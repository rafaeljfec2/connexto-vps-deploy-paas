/* eslint-disable react-refresh/only-export-components */
import {
  type ReactNode,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { API_ROUTES, ROUTES } from "@/constants/routes";
import { api } from "@/services/api";

export type AuthProvider = "email" | "github";
export type UserRole = "admin" | "member";

export interface User {
  readonly id: string;
  readonly githubId?: number;
  readonly githubLogin?: string;
  readonly name: string;
  readonly email: string;
  readonly avatarUrl?: string;
  readonly authProvider: AuthProvider;
  readonly role: UserRole;
  readonly createdAt: string;
}

interface AuthContextType {
  readonly user: User | null;
  readonly isLoading: boolean;
  readonly isAuthenticated: boolean;
  readonly isAdmin: boolean;
  readonly login: () => void;
  readonly logout: () => Promise<void>;
  readonly refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthProviderProps {
  readonly children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchUser = useCallback(async () => {
    try {
      const userData = await api.auth.me();
      setUser(userData);
    } catch {
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUser();
  }, [fetchUser]);

  const login = useCallback(() => {
    globalThis.location.href = API_ROUTES.AUTH.GITHUB;
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.auth.logout();
    } finally {
      setUser(null);
      globalThis.location.href = ROUTES.LOGIN;
    }
  }, []);

  const refresh = useCallback(async () => {
    setIsLoading(true);
    await fetchUser();
  }, [fetchUser]);

  const isAuthenticated = user !== null;
  const isAdmin = user?.role === "admin";

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isLoading,
      isAuthenticated,
      isAdmin,
      login,
      logout,
      refresh,
    }),
    [user, isLoading, isAuthenticated, isAdmin, login, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
