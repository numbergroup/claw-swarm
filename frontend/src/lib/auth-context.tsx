"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import type { User } from "./types";
import * as api from "./api";

function getTokenExpiry(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    const decoded = JSON.parse(atob(payload));
    return typeof decoded.exp === "number" ? decoded.exp * 1000 : null;
  } catch {
    return null;
  }
}

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
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const scheduleRefresh = useCallback((jwt: string) => {
    if (refreshTimer.current) clearTimeout(refreshTimer.current);
    const expiry = getTokenExpiry(jwt);
    if (!expiry) return;
    const delay = expiry - Date.now() - 5 * 60 * 1000; // 5 minutes before expiry
    if (delay <= 0) return;
    refreshTimer.current = setTimeout(async () => {
      try {
        const res = await api.refreshToken();
        localStorage.setItem("token", res.token);
        setToken(res.token);
        setUser(res.user);
        scheduleRefresh(res.token);
      } catch {
        localStorage.removeItem("token");
        setToken(null);
        setUser(null);
      }
    }, delay);
  }, []);

  useEffect(() => {
    const stored = localStorage.getItem("token");
    if (stored) {
      setToken(stored);
    } else {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!token) return;
    api
      .getMe()
      .then((u) => {
        setUser(u);
        scheduleRefresh(token);
      })
      .catch(() => {
        localStorage.removeItem("token");
        setToken(null);
      })
      .finally(() => setLoading(false));
  }, [token, scheduleRefresh]);

  useEffect(() => {
    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current);
    };
  }, []);

  const loginFn = useCallback(
    async (email: string, password: string) => {
      const res = await api.login({ email, password });
      localStorage.setItem("token", res.token);
      setToken(res.token);
      setUser(res.user);
      scheduleRefresh(res.token);
      router.push("/");
    },
    [router, scheduleRefresh],
  );

  const signupFn = useCallback(
    async (email: string, password: string, displayName?: string) => {
      const res = await api.signup({ email, password, displayName });
      localStorage.setItem("token", res.token);
      setToken(res.token);
      setUser(res.user);
      scheduleRefresh(res.token);
      router.push("/");
    },
    [router, scheduleRefresh],
  );

  const logout = useCallback(() => {
    if (refreshTimer.current) clearTimeout(refreshTimer.current);
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
