import { useEffect, useMemo, useState } from "react";
import {
  ApiError,
  createOrder,
  generateIdempotencyKey,
  getCart,
  listAddresses,
  type AddressList,
  type Address,
  type CartResponse,
} from "../api";
import { useAuth } from "../auth";
import { ui } from "../ui/styles";

function isAddressList(value: unknown): value is AddressList {
  if (typeof value !== "object" || value === null) return false;
  if (!("items" in value)) return false;
  const items = (value as { items: unknown }).items;
  return Array.isArray(items);
}

function isAddressArray(value: unknown): value is Address[] {
  return Array.isArray(value);
}

function formatYen(n: number): string {
  return `¥${n.toLocaleString("ja-JP")}`;
}

export default function CheckoutPage() {
  const { accessToken } = useAuth();

  const [addresses, setAddresses] = useState<AddressList | null>(null);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const [cart, setCart] = useState<CartResponse | null>(null);

  const [error, setError] = useState<string>("");
  const [msg, setMsg] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const addressItems: Address[] = useMemo(() => {
    return addresses?.items ?? [];
  }, [addresses]);

  const defaultAddressId = useMemo<number | null>(() => {
    const d = addressItems.find((x) => x.is_default);
    return d ? d.id : null;
  }, [addressItems]);

  const cartItems = useMemo(() => cart?.items ?? [], [cart]);
  const isCartEmpty = cartItems.length === 0;

  // 住所とカートを並行で
  useEffect(() => {
    let alive = true;

    async function load(): Promise<void> {
      if (!accessToken) return;

      setIsLoading(true);
      setError("");

      try {
        const [addrRes, cartRes] = await Promise.all([
          listAddresses(accessToken),
          getCart(accessToken),
        ]);

        if (!alive) return;

        // addresses
        if (isAddressList(addrRes)) {
          setAddresses(addrRes);
          setSelectedId((prev) => {
            const defaultId =
              addrRes.items.find((x) => x.is_default)?.id ?? null;
            return prev ?? defaultId;
          });
        } else if (isAddressArray(addrRes)) {
          const list: AddressList = { items: addrRes };
          setAddresses(list);
          setSelectedId((prev) => {
            const defaultId = list.items.find((x) => x.is_default)?.id ?? null;
            return prev ?? defaultId;
          });
        } else {
          setAddresses({ items: [] });
          setSelectedId(null);
          setError(
            "住所一覧のレスポンス形式が想定と違います（items が見つかりません）",
          );
        }

        // cart
        setCart(cartRes);
      } catch (e: unknown) {
        if (!alive) return;
        setAddresses({ items: [] });
        setSelectedId(null);
        setCart({ items: [], total: 0 });
        setError(e instanceof ApiError ? e.message : "予期せぬエラー");
      } finally {
        if (alive) setIsLoading(false);
      }
    }

    void load();

    return () => {
      alive = false;
    };
  }, [accessToken]);

  async function onOrder(): Promise<void> {
    if (!accessToken) return;

    setError("");
    setMsg("");

    // カートが空なら注文できない（UI的にも自然）
    if (!cart || cart.items.length === 0) {
      setError("カートが空です。先に商品をカートに追加してください。");
      return;
    }

    const addressId: number | null = selectedId ?? defaultAddressId;
    if (!addressId) {
      setError("住所が選択されていません（先に住所を登録してください）");
      return;
    }

    const idempotencyKey: string = generateIdempotencyKey();

    setIsSubmitting(true);
    try {
      await createOrder(accessToken, addressId, idempotencyKey);
      setMsg("商品を注文しました");

      // 注文後はカートがクリアされるので、表示も更新（空にする）
      setCart({ items: [], total: 0 });
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
          <h2 style={ui.title}>会計</h2>
          <p style={ui.subtitle}>注文を確定するにはログインが必要です。</p>
        </div>

        <div style={ui.card}>
          <p style={ui.msgErr}>ログインが必要です</p>
        </div>
      </div>
    );
  }

  return (
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>会計</h2>
        <p style={ui.subtitle}>注文内容と配送先を確認して確定します。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}
        {msg ? <p style={ui.msgOk}>{msg}</p> : null}

        {isLoading && (!addresses || !cart) ? (
          <p style={ui.hint}>loading...</p>
        ) : null}

        {cart ? (
          <div style={ui.productCard}>
            <div style={{ ...ui.productName, marginBottom: 8 }}>注文内容</div>

            {isCartEmpty ? (
              <p style={ui.hint}>カートが空です</p>
            ) : (
              <div style={{ display: "grid", gap: 10 }}>
                <div style={{ display: "grid", gap: 8 }}>
                  {cartItems.map((it) => {
                    const lineTotal: number = it.price * it.quantity;
                    return (
                      <div
                        key={it.id}
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
                            {it.quantity}個（{formatYen(it.price)}）
                          </div>
                        </div>
                        <div style={{ fontWeight: 900 }}>
                          {formatYen(lineTotal)}
                        </div>
                      </div>
                    );
                  })}
                </div>

                <div
                  style={{ display: "flex", justifyContent: "space-between" }}
                >
                  <span style={{ fontWeight: 900 }}>合計</span>
                  <span style={{ fontWeight: 900 }}>
                    {formatYen(cart.total)}
                  </span>
                </div>
              </div>
            )}
          </div>
        ) : null}

        {addresses ? (
          <div style={{ display: "grid", gap: 12, marginTop: 12 }}>
            <div style={ui.productCard}>
              <div style={{ ...ui.productName, marginBottom: 8 }}>配送先</div>

              <label style={ui.label}>
                住所
                <select
                  value={selectedId ?? ""}
                  onChange={(e) =>
                    setSelectedId(
                      e.target.value ? Number(e.target.value) : null,
                    )
                  }
                  style={ui.select}
                  disabled={isSubmitting}
                >
                  <option value="">(select)</option>
                  {addressItems.map((a) => (
                    <option key={a.id} value={a.id}>
                      #{a.id} {a.name} {a.is_default ? "(default)" : ""}
                    </option>
                  ))}
                </select>
              </label>

              {addressItems.length === 0 ? (
                <p style={ui.hint}>
                  住所がありません。先に「住所」ページで住所を登録してください。
                </p>
              ) : null}
            </div>

            <button
              onClick={() => void onOrder()}
              disabled={
                isSubmitting || addressItems.length === 0 || isCartEmpty
              }
              style={{
                ...ui.buttonPrimary,
                ...(isSubmitting || addressItems.length === 0 || isCartEmpty
                  ? ui.buttonPrimaryDisabled
                  : null),
              }}
            >
              注文を確定する
            </button>

            {isCartEmpty ? (
              <p style={ui.hint}>
                注文するには、先にカートに商品を追加してください。
              </p>
            ) : null}
          </div>
        ) : (
          <p style={ui.hint}>loading...</p>
        )}
      </div>
    </div>
  );
}
