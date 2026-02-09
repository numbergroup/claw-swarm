"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import type { User } from "./types";
import * as api from "./api";

interface AuthContextValue {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, displayName?: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const initialToken = typeof window === "undefined" ? null : localStorage.getItem("token");
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(initialToken);
  const [loading, setLoading] = useState(initialToken !== null);
  const router = useRouter();

  useEffect(() => {
    if (!token) return;
    api
      .getMe()
      .then((u) => {
        setUser(u);
      })
      .catch(() => {
        localStorage.removeItem("token");
        setToken(null);
      })
      .finally(() => setLoading(false));
  }, [token]);

  const loginFn = useCallback(
    async (email: string, password: string) => {
      const res = await api.login({ email, password });
      localStorage.setItem("token", res.token);
      setToken(res.token);
      setUser(res.user);
      router.push("/");
    },
    [router],
  );

  const signupFn = useCallback(
    async (email: string, password: string, displayName?: string) => {
      const res = await api.signup({ email, password, displayName });
      localStorage.setItem("token", res.token);
      setToken(res.token);
      setUser(res.user);
      router.push("/");
    },
    [router],
  );

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    setToken(null);
    setUser(null);
    router.push("/login");
  }, [router]);

  return (
    <AuthContext.Provider
      value={{ user, token, loading, login: loginFn, signup: signupFn, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
