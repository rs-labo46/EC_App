import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";

export default function NavBar() {
  const { user, accessToken, logout, fetchMe, isLoading } = useAuth();

  useEffect(() => {
    // accessToken が入ったタイミングで1回だけ /me を取りに行く
    if (!accessToken) return;
    void fetchMe().catch(() => {});
  }, [accessToken]);

  return (
    <div
      style={{
        display: "flex",
        gap: 12,
        padding: 12,
        borderBottom: "1px solid #333",
      }}
    >
      <Link to="/">商品</Link>

      {user?.role === "ADMIN" ? (
        <Link to="/admin/products/new">商品作成</Link>
      ) : null}

      <Link to="/cart">カート</Link>
      <Link to="/addresses">住所</Link>
      <Link to="/orders">注文</Link>

      <div
        style={{
          marginLeft: "auto",
          display: "flex",
          gap: 12,
          alignItems: "center",
        }}
      >
        <span>{user ? `${user.email} (${user.role})` : "Guest"}</span>

        {accessToken ? (
          <button onClick={() => void logout()} disabled={isLoading}>
            ログアウト
          </button>
        ) : (
          <>
            <Link to="/login">ログイン</Link>
            <Link to="/signup">サインアップ</Link>
          </>
        )}
      </div>
    </div>
  );
}
