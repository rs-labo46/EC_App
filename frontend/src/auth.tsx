// 認証状態（access token + user）をまとめて管理
import React, {
  createContext,
  useContext,
  useMemo,
  useState,
  type JSX,
} from "react";
import {
  authLogin,
  authLogout,
  authRefresh,
  getMe,
  type User,
  type JwtAccessToken,
  ApiError,
} from "./api";

type AuthState = {
  accessToken: string | null;
  user: User | null;
  isLoading: boolean;
};

type AuthContextValue = AuthState & {
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshAndSetToken: () => Promise<void>;
  fetchMe: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider(props: {
  children: React.ReactNode;
}): JSX.Element {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  async function login(email: string, password: string): Promise<void> {
    setIsLoading(true);
    try {
      const res = await authLogin(email, password);
      setAccessToken(res.token.access_token);
      setUser(res.user);
    } finally {
      setIsLoading(false);
    }
  }

  async function logout(): Promise<void> {
    if (!accessToken) {
      setUser(null);
      return;
    }
    setIsLoading(true);
    try {
      await authLogout(accessToken);
      setAccessToken(null);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }

  //  refreshを呼び、accesstokenを更新する

  async function refreshAndSetToken(): Promise<void> {
    setIsLoading(true);
    try {
      const token: JwtAccessToken = await authRefresh();
      setAccessToken(token.access_token);
      //userは/meで取る
    } finally {
      setIsLoading(false);
    }
  }

  // /me を叩いて user を更新する

  async function fetchMe(): Promise<void> {
    if (!accessToken) return;

    setIsLoading(true);
    try {
      const me = await getMe(accessToken);
      setUser(me);
      return;
    } catch (e: unknown) {
      //401ならrefresh→/me再試行
      if (e instanceof ApiError && e.status === 401) {
        try {
          const token = await authRefresh();
          setAccessToken(token.access_token);
          const me = await getMe(token.access_token);
          setUser(me);
          return;
        } catch {
          // refresh できないならログアウト扱い
          setAccessToken(null);
          setUser(null);
          return;
        }
      }
      throw e;
    } finally {
      setIsLoading(false);
    }
  }

  const value = useMemo<AuthContextValue>(
    () => ({
      accessToken,
      user,
      isLoading,
      login,
      logout,
      refreshAndSetToken,
      fetchMe,
    }),
    [accessToken, user, isLoading],
  );

  return (
    <AuthContext.Provider value={value}>{props.children}</AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
