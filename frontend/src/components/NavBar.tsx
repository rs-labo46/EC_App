import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";
import { ui } from "../ui/styles";

export default function NavBar() {
  const { user, accessToken, logout, fetchMe, isLoading } = useAuth();

  useEffect(() => {
    // accessToken が入ったタイミングで1回だけ /me を取りに行く
    if (!accessToken) return;
    void fetchMe().catch(() => {});
  }, [accessToken]);

  return (
    <div style={ui.navWrap}>
      <div style={ui.navInner}>
        <Link to="/" style={ui.navLink}>
          商品
        </Link>

        {user?.role === "ADMIN" ? (
          <Link to="/admin/products/new" style={ui.navLink}>
            商品作成
          </Link>
        ) : null}

        <Link to="/cart" style={ui.navLink}>
          カート
        </Link>
        <Link to="/addresses" style={ui.navLink}>
          住所
        </Link>
        <Link to="/orders" style={ui.navLink}>
          注文
        </Link>

        <div style={ui.navRight}>
          <span style={{ color: "rgba(255,255,255,0.80)", fontSize: 13 }}>
            {user ? `${user.email} (${user.role})` : "Guest"}
          </span>

          {accessToken ? (
            <button
              onClick={() => void logout()}
              disabled={isLoading}
              style={{
                padding: "10px 14px",
                borderRadius: 12,
                border: "1px solid rgba(255,255,255,0.12)",
                background: "rgba(0,0,0,0.18)",
                color: "rgba(255,255,255,0.92)",
                fontWeight: 800,
                cursor: isLoading ? "not-allowed" : "pointer",
                opacity: isLoading ? 0.6 : 1,
              }}
            >
              ログアウト
            </button>
          ) : (
            <>
              <Link to="/login" style={ui.navLink}>
                ログイン
              </Link>
              <Link to="/signup" style={ui.navLink}>
                サインアップ
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
