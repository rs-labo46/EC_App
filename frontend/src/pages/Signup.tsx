import { Link, useNavigate } from "react-router-dom";
import { ApiError, authRegister } from "../api";
import { useState } from "react";

export default function SignupPage() {
  const nav = useNavigate();

  const [email, setEmail] = useState<string>("user2@test.com");
  const [password, setPassword] = useState<string>("CorrectPW123!");
  const [error, setError] = useState<string>("");
  const [msg, setMsg] = useState<string>("");

  async function onSubmit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    setError("");
    setMsg("");

    try {
      const res = await authRegister(email, password);
      setMsg(res.message ?? "registered");
      nav("/login", { replace: true });
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        return;
      }
      setError("unexpected error");
    }
  }

  return (
    <div style={{ maxWidth: 420 }}>
      <h2>サインアップ</h2>

      <form
        onSubmit={(e) => void onSubmit(e)}
        style={{ display: "grid", gap: 8 }}
      >
        <label>
          Eメール
          <input
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={{ width: "100%" }}
          />
        </label>

        <label>
          パスワード
          <input
            value={password}
            type="password"
            onChange={(e) => setPassword(e.target.value)}
            style={{ width: "100%" }}
          />
        </label>

        <button type="submit">アカウントを作成</button>
      </form>

      {msg ? <p style={{ color: "lime" }}>{msg}</p> : null}
      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}

      <p style={{ fontSize: 12, opacity: 0.8 }}>
        すでにアカウントがある場合は <Link to="/login">Login</Link>
      </p>
    </div>
  );
}
