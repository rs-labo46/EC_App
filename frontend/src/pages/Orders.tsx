import { useEffect, useState } from "react";
import { ApiError, listOrders, type Order } from "../api";
import { useAuth } from "../auth";
import { Link } from "react-router-dom";
import { ui } from "../ui/styles";

export default function OrdersPage() {
  const { accessToken } = useAuth();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let alive = true;
    async function load(): Promise<void> {
      if (!accessToken) return;
      setError("");
      try {
        const res = await listOrders(accessToken);
        if (!alive) return;
        setOrders(res);
      } catch (e: unknown) {
        if (!alive) return;
        setError(e instanceof ApiError ? e.message : "unexpected error");
      }
    }
    void load();
    return () => {
      alive = false;
    };
  }, [accessToken]);

  return (
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>Orders</h2>
        <p style={ui.subtitle}>過去の注文を確認できます。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}

        {!orders ? (
          <p style={ui.hint}>loading...</p>
        ) : orders.length === 0 ? (
          <p style={ui.hint}>注文はまだありません。</p>
        ) : (
          <div style={ui.grid}>
            {orders.map((o) => (
              <div key={o.id} style={ui.productCard}>
                <p style={ui.productName}>
                  <Link to={`/orders/${o.id}`}>#{o.id}</Link>
                </p>
                <div style={ui.hint}>ステータス: {o.status}</div>
                <div style={ui.price}>¥{o.total_price}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
