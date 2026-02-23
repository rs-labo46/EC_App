import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { ApiError, getOrder, type Order } from "../api";
import { useAuth } from "../auth";

export default function OrderDetailPage() {
  const { id } = useParams();
  const orderId: number = Number(id);
  const { accessToken } = useAuth();

  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let alive = true;
    async function load(): Promise<void> {
      if (!accessToken) return;
      setError("");
      try {
        const res = await getOrder(accessToken, orderId);
        if (!alive) return;
        setOrder(res);
      } catch (e: unknown) {
        if (!alive) return;
        setError(e instanceof ApiError ? e.message : "unexpected error");
      }
    }
    if (!Number.isFinite(orderId)) return;
    void load();
    return () => {
      alive = false;
    };
  }, [accessToken, orderId]);

  if (!Number.isFinite(orderId)) return <p>invalid id</p>;

  return (
    <div>
      <h2>注文詳細</h2>
      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {!order ? (
        <p>loading...</p>
      ) : (
        <div style={{ display: "grid", gap: 8 }}>
          <div>
            <b>#{order.id}</b> — {order.status}
          </div>
          <div>合計: ¥{order.total_price}</div>
          <div>作成日: {order.created_at}</div>
          <h3>商品一覧</h3>
          <ul>
            {order.items.map((it, idx) => (
              <li key={`${it.product_id}_${idx}`}>
                {it.name} — ¥{it.price} × {it.quantity}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
