import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { ApiError, getOrder, type Order } from "../api";
import { useAuth } from "../auth";
import { ui } from "../ui/styles";

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
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>注文詳細</h2>
        <p style={ui.subtitle}>注文番号・合計・商品明細を確認できます。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}

        {!order ? (
          <p style={ui.hint}>loading...</p>
        ) : (
          <div style={{ display: "grid", gap: 12 }}>
            <div style={ui.productCard}>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 12,
                }}
              >
                <div style={{ ...ui.productName, fontSize: 18 }}>
                  #{order.id}
                </div>
                <div style={ui.hint}>{order.status}</div>
              </div>

              <div style={{ marginTop: 10, display: "grid", gap: 6 }}>
                <div>
                  <span style={ui.hint}>合計</span>
                  <div style={ui.price}>¥{order.total_price}</div>
                </div>
                <div style={ui.hint}>作成日: {order.created_at}</div>
              </div>
            </div>

            <div style={ui.productCard}>
              <div style={{ ...ui.productName, marginBottom: 8 }}>商品一覧</div>

              {order.items.length === 0 ? (
                <p style={ui.hint}>商品がありません。</p>
              ) : (
                <div style={{ display: "grid", gap: 8 }}>
                  {order.items.map((it, idx) => (
                    <div
                      key={`${it.product_id}_${idx}`}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        gap: 12,
                        padding: "10px 12px",
                        borderRadius: 12,
                        border: "1px solid rgba(255,255,255,0.10)",
                        background: "rgba(0,0,0,0.18)",
                      }}
                    >
                      <div>
                        <div style={{ fontWeight: 800 }}>{it.name}</div>
                        <div style={ui.hint}>
                          ¥{it.price} × {it.quantity}
                        </div>
                      </div>
                      <div style={{ fontWeight: 900 }}>
                        ¥{it.price * it.quantity}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
