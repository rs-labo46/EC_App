import { useEffect, useState } from "react";
import { ApiError, listOrders, type Order } from "../api";
import { useAuth } from "../auth";
import { Link } from "react-router-dom";

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
    <div>
      <h2>Orders</h2>
      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {!orders ? (
        <p>loading...</p>
      ) : (
        <ul>
          {orders.map((o) => (
            <li key={o.id}>
              <Link to={`/orders/${o.id}`}>#{o.id}</Link> — {o.status} — ¥
              {o.total_price}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
