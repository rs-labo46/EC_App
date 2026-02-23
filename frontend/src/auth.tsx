import React, {
  createContext,
  useContext,
  useMemo,
  useState,
  useEffect,
  useRef,
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

  // 初回マウント時に「1回だけ」復元処理を走らせる
  const bootRef = useRef<boolean>(false);

  useEffect(() => {
    if (bootRef.current) return;
    bootRef.current = true;

    async function bootstrap(): Promise<void> {
      try {
        // refresh cookie + CSRF が揃っていれば accessToken が再発行される
        const token: JwtAccessToken = await authRefresh();
        setAccessToken(token.access_token);

        // token だけでは user が分からないので /me を叩く
        const me: User = await getMe(token.access_token);
        setUser(me);
      } catch {
        // refreshできない＝未ログイン扱いでOK
        setAccessToken(null);
        setUser(null);
      }
    }

    void bootstrap();
  }, []);

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

  // refreshを呼び、accessTokenを更新する
  async function refreshAndSetToken(): Promise<void> {
    setIsLoading(true);
    try {
      const token: JwtAccessToken = await authRefresh();
      setAccessToken(token.access_token);
    } finally {
      setIsLoading(false);
    }
  }

  // /meを叩いて user を更新する（401なら refresh→/me 再試行）
  async function fetchMe(): Promise<void> {
    if (!accessToken) return;

    setIsLoading(true);
    try {
      const me = await getMe(accessToken);
      setUser(me);
      return;
    } catch (e: unknown) {
      if (e instanceof ApiError && e.status === 401) {
        try {
          const token = await authRefresh();
          setAccessToken(token.access_token);
          const me = await getMe(token.access_token);
          setUser(me);
          return;
        } catch {
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
