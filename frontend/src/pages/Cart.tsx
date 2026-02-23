import { useEffect, useMemo, useState } from "react";
import {
  ApiError,
  deleteCartItem,
  getCart,
  updateCartItem,
  type CartResponse,
} from "../api";
import { useAuth } from "../auth";
import { useNavigate } from "react-router-dom";

export default function CartPage() {
  const { accessToken } = useAuth();
  const nav = useNavigate();

  const [cart, setCart] = useState<CartResponse | null>(null);
  const [error, setError] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const items = useMemo(() => cart?.items ?? [], [cart]);
  const isEmpty = items.length === 0;

  async function load(): Promise<void> {
    if (!accessToken) return;

    setIsLoading(true);
    setError("");

    try {
      const c = await getCart(accessToken);
      setCart(c);
    } catch (e: unknown) {
      setCart({ items: [], total: 0 });
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [accessToken]);

  async function onChangeQty(cartItemId: number, qty: number): Promise<void> {
    if (!accessToken) return;
    if (!Number.isFinite(qty) || qty < 1) {
      setError("数量は1以上にしてください");
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      const c = await updateCartItem(accessToken, cartItemId, qty);
      setCart(c);
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function onDelete(cartItemId: number): Promise<void> {
    if (!accessToken) return;

    setIsSubmitting(true);
    setError("");

    try {
      const c = await deleteCartItem(accessToken, cartItemId);
      setCart(c);
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div style={{ display: "grid", gap: 12 }}>
      <h2>Cart</h2>

      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}

      {isLoading && !cart ? <p>loading...</p> : null}

      {cart ? (
        isEmpty ? (
          <div style={{ display: "grid", gap: 8 }}>
            <p style={{ opacity: 0.85 }}>カートが空です</p>
            <button onClick={() => nav("/")}>商品一覧へ</button>
          </div>
        ) : (
          <div style={{ display: "grid", gap: 10 }}>
            <ul>
              {items.map((it) => (
                <li key={it.id} style={{ marginBottom: 8 }}>
                  {it.name} — ¥{it.price} ×{" "}
                  <input
                    type="number"
                    min={1}
                    value={it.quantity}
                    onChange={(e) =>
                      void onChangeQty(it.id, Number(e.target.value))
                    }
                    style={{ width: 70 }}
                    disabled={isSubmitting}
                  />{" "}
                  <button
                    onClick={() => void onDelete(it.id)}
                    disabled={isSubmitting}
                  >
                    削除
                  </button>
                </li>
              ))}
            </ul>

            <div>
              <b>合計:</b> ¥{cart.total}
            </div>

            <div style={{ display: "flex", gap: 8 }}>
              <button onClick={() => nav("/checkout")} disabled={isSubmitting}>
                Checkoutへ進む
              </button>
              <button onClick={() => nav("/")}>買い物を続ける</button>
            </div>
          </div>
        )
      ) : (
        <p>loading...</p>
      )}
    </div>
  );
}
