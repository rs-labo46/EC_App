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
import { ui } from "../ui/styles";

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
    if (!accessToken) {
      setCart(null);
      return;
    }

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
    // accessTokenが変わったら再ロード（ログイン/ログアウト/refresh後）
  }, [accessToken]);

  function toSafeQty(raw: string): number | null {
    if (raw.trim() === "") return null;
    const n: number = Number(raw);
    if (!Number.isFinite(n)) return null;
    return n;
  }

  async function onChangeQty(
    cartItemId: number,
    nextQty: number,
  ): Promise<void> {
    if (!accessToken) return;

    if (!Number.isFinite(nextQty) || nextQty < 1) {
      setError("数量は1以上にしてください");
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      const c = await updateCartItem(accessToken, cartItemId, nextQty);
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

  if (!accessToken) {
    return (
      <div style={ui.page}>
        <div style={ui.header}>
          <h2 style={ui.title}>Cart</h2>
          <p style={ui.subtitle}>カートを見るにはログインが必要です。</p>
        </div>

        <div style={ui.card}>
          <p style={ui.hint}>カートを見るにはログインが必要です</p>
          <div
            style={{
              display: "flex",
              gap: 10,
              marginTop: 12,
              flexWrap: "wrap",
            }}
          >
            <button style={ui.buttonPrimary} onClick={() => nav("/login")}>
              ログインへ
            </button>
            <button style={ui.buttonPrimary} onClick={() => nav("/")}>
              商品一覧へ
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>Cart</h2>
        <p style={ui.subtitle}>数量の変更・削除・会計へ進めます。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}
        {isLoading && !cart ? <p style={ui.hint}>loading...</p> : null}

        {cart ? (
          isEmpty ? (
            <div style={ui.productCard}>
              <p style={ui.hint}>カートが空です</p>
              <button style={ui.buttonPrimary} onClick={() => nav("/")}>
                商品一覧へ
              </button>
            </div>
          ) : (
            <div style={{ display: "grid", gap: 12 }}>
              <div style={{ display: "grid", gap: 10 }}>
                {items.map((it) => (
                  <div
                    key={it.id}
                    style={{
                      ...ui.productCard,
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      gap: 12,
                    }}
                  >
                    <div>
                      <div style={{ fontWeight: 900 }}>{it.name}</div>
                      <div style={ui.hint}>¥{it.price}</div>
                    </div>

                    <div
                      style={{ display: "flex", gap: 8, alignItems: "center" }}
                    >
                      <input
                        type="number"
                        min={1}
                        value={it.quantity}
                        onChange={(e) => {
                          const parsed: number | null = toSafeQty(
                            e.target.value,
                          );
                          if (parsed === null) return;
                          void onChangeQty(it.id, parsed);
                        }}
                        style={{ ...ui.input, width: 90 }}
                        disabled={isSubmitting}
                      />

                      <button
                        onClick={() => void onDelete(it.id)}
                        disabled={isSubmitting}
                        style={{
                          ...ui.buttonPrimary,
                          ...(isSubmitting ? ui.buttonPrimaryDisabled : null),
                        }}
                      >
                        削除
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              <div style={ui.productCard}>
                <div
                  style={{ display: "flex", justifyContent: "space-between" }}
                >
                  <span style={{ fontWeight: 900 }}>合計</span>
                  <span style={{ fontWeight: 900 }}>¥{cart.total}</span>
                </div>
              </div>

              <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                <button
                  onClick={() => nav("/checkout")}
                  disabled={isSubmitting}
                  style={{
                    ...ui.buttonPrimary,
                    ...(isSubmitting ? ui.buttonPrimaryDisabled : null),
                  }}
                >
                  Checkoutへ進む
                </button>
                <button style={ui.buttonPrimary} onClick={() => nav("/")}>
                  買い物を続ける
                </button>
              </div>
            </div>
          )
        ) : (
          <p style={ui.hint}>loading...</p>
        )}
      </div>
    </div>
  );
}
