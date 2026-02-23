import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../api";
import { useAuth } from "../auth";

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
    <div style={{ maxWidth: 420 }}>
      <h2>Login</h2>
      <form
        onSubmit={(e) => void onSubmit(e)}
        style={{ display: "grid", gap: 8 }}
      >
        <label>
          Email
          <input
            placeholder="user1@test.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={{ width: "100%" }}
          />
        </label>
        <label>
          Password
          <input
            placeholder="CorrectPW123!"
            value={password}
            type="password"
            onChange={(e) => setPassword(e.target.value)}
            style={{ width: "100%" }}
          />
        </label>
        <button disabled={isLoading} type="submit">
          ログイン
        </button>
      </form>
      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      <p style={{ opacity: 0.8, fontSize: 12 }}></p>
    </div>
  );
}
