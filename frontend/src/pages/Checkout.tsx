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
      <div style={{ display: "grid", gap: 12, maxWidth: 560 }}>
        <h2>会計</h2>
        <p style={{ color: "tomato" }}>ログインが必要です</p>
      </div>
    );
  }

  return (
    <div style={{ display: "grid", gap: 12, maxWidth: 560 }}>
      <h2>会計</h2>

      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {msg ? <p style={{ color: "lime" }}>{msg}</p> : null}

      {isLoading && (!addresses || !cart) ? <p>loading...</p> : null}

      {/* カート明細 */}
      {cart ? (
        <div
          style={{
            display: "grid",
            gap: 8,
            padding: 12,
            border: "1px solid #ddd",
          }}
        >
          <b>注文内容</b>

          {isCartEmpty ? (
            <p style={{ opacity: 0.85 }}>カートが空です</p>
          ) : (
            <div style={{ display: "grid", gap: 6 }}>
              <ul style={{ margin: 0, paddingLeft: 18 }}>
                {cartItems.map((it) => {
                  const lineTotal: number = it.price * it.quantity;
                  return (
                    <li key={it.id}>
                      {it.name} × {it.quantity}（{formatYen(it.price)}）＝{" "}
                      {formatYen(lineTotal)}
                    </li>
                  );
                })}
              </ul>

              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span>
                  <b>合計</b>
                </span>
                <span>
                  <b>{formatYen(cart.total)}</b>
                </span>
              </div>
            </div>
          )}
        </div>
      ) : null}

      {/* 住所 + 注文確定 */}
      {addresses ? (
        <div style={{ display: "grid", gap: 8 }}>
          <label>
            住所:
            <select
              value={selectedId ?? ""}
              onChange={(e) =>
                setSelectedId(e.target.value ? Number(e.target.value) : null)
              }
              style={{ marginLeft: 8 }}
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
            <p style={{ opacity: 0.8 }}>
              住所がありません。先に「住所」ページで住所を登録してください。
            </p>
          ) : null}

          <button
            onClick={() => void onOrder()}
            disabled={isSubmitting || addressItems.length === 0 || isCartEmpty}
          >
            注文を確定する
          </button>

          {isCartEmpty ? (
            <p style={{ opacity: 0.8 }}>
              注文するには、先にカートに商品を追加してください。
            </p>
          ) : null}
        </div>
      ) : (
        <p>loading...</p>
      )}
    </div>
  );
}
