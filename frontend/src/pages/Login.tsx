import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../api";
import { useAuth } from "../auth";
import { ui } from "../ui/styles";

export default function LoginPage() {
  const { login, isLoading } = useAuth();
  const nav = useNavigate();

  const [email, setEmail] = useState<string>("aaa@test.com");
  const [password, setPassword] = useState<string>("ASDasd123");
  const [error, setError] = useState<string>("");

  async function onSubmit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    setError("");
    try {
      await login(email, password);
      nav("/", { replace: true });
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.body?.error ?? err.message);
        return;
      }
      setError("unexpected error");
    }
  }

  return (
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>Login</h2>
        <p style={ui.subtitle}>ログインするとカートに追加して注文できます。</p>
      </div>

      <div style={ui.card}>
        <form onSubmit={(e) => void onSubmit(e)} style={ui.form}>
          <label style={ui.label}>
            Email
            <input
              placeholder="user1@test.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              style={ui.input}
              autoComplete="email"
            />
          </label>

          <label style={ui.label}>
            Password
            <input
              placeholder="CorrectPW123!"
              value={password}
              type="password"
              onChange={(e) => setPassword(e.target.value)}
              style={ui.input}
              autoComplete="current-password"
            />
          </label>

          <button
            disabled={isLoading}
            type="submit"
            style={{
              ...ui.buttonPrimary,
              ...(isLoading ? ui.buttonPrimaryDisabled : null),
            }}
          >
            ログイン
          </button>
        </form>

        {error ? <p style={ui.msgErr}>{error}</p> : null}
      </div>
    </div>
  );
}
