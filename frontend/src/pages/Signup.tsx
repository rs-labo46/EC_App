import { Link, useNavigate } from "react-router-dom";
import { ApiError, authRegister } from "../api";
import { useState } from "react";
import { ui } from "../ui/styles";

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
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>サインアップ</h2>
        <p style={ui.subtitle}>
          アカウントを作成して、カート・注文機能を使えるようにします。
        </p>
      </div>

      <div style={ui.card}>
        <form onSubmit={(e) => void onSubmit(e)} style={ui.form}>
          <label style={ui.label}>
            Eメール
            <input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              style={ui.input}
              autoComplete="email"
            />
          </label>

          <label style={ui.label}>
            パスワード
            <input
              value={password}
              type="password"
              onChange={(e) => setPassword(e.target.value)}
              style={ui.input}
              autoComplete="new-password"
            />
          </label>

          <button type="submit" style={ui.buttonPrimary}>
            アカウントを作成
          </button>
        </form>

        {msg ? <p style={ui.msgOk}>{msg}</p> : null}
        {error ? <p style={ui.msgErr}>{error}</p> : null}

        <p style={ui.footnote}>
          すでにアカウントがある場合は <Link to="/login">Login</Link>
        </p>
      </div>
    </div>
  );
}
