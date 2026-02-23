import { useEffect, useMemo, useState } from "react";
import {
  ApiError,
  createOrder,
  generateIdempotencyKey,
  listAddresses,
  type AddressList,
  type Address,
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

export default function CheckoutPage() {
  const { accessToken } = useAuth();

  const [addresses, setAddresses] = useState<AddressList | null>(null);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [error, setError] = useState<string>("");
  const [msg, setMsg] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const items: Address[] = useMemo(() => {
    return addresses?.items ?? [];
  }, [addresses]);

  const defaultAddressId = useMemo<number | null>(() => {
    const d = items.find((x) => x.is_default);
    return d ? d.id : null;
  }, [items]);

  useEffect(() => {
    let alive = true;

    async function load(): Promise<void> {
      if (!accessToken) return;

      setIsLoading(true);
      setError("");

      try {
        const res: unknown = await listAddresses(accessToken);
        if (!alive) return;

        if (isAddressList(res)) {
          setAddresses(res);
          setSelectedId((prev) => {
            const defaultId = res.items.find((x) => x.is_default)?.id ?? null;
            return prev ?? defaultId;
          });
          return;
        }

        // backendが配列で返しても落ちないように吸収
        if (isAddressArray(res)) {
          const list: AddressList = { items: res };
          setAddresses(list);
          setSelectedId((prev) => {
            const defaultId = list.items.find((x) => x.is_default)?.id ?? null;
            return prev ?? defaultId;
          });
          return;
        }

        // 想定外の形
        setAddresses({ items: [] });
        setSelectedId(null);
        setError(
          "住所一覧のレスポンス形式が想定と違います（items が見つかりません）",
        );
      } catch (e: unknown) {
        if (!alive) return;
        setAddresses({ items: [] });
        setSelectedId(null);
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

    const addressId: number | null = selectedId ?? defaultAddressId;
    if (!addressId) {
      setError("住所が選択されていません（先に住所を登録してください）");
      return;
    }

    const idempotencyKey: string = generateIdempotencyKey();

    setIsSubmitting(true);
    try {
      const order = await createOrder(accessToken, addressId, idempotencyKey);
      setMsg(`order created: #${order.id} (${order.status})`);
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

      {isLoading && !addresses ? <p>loading...</p> : null}

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
              {items.map((a) => (
                <option key={a.id} value={a.id}>
                  #{a.id} {a.name} {a.is_default ? "(default)" : ""}
                </option>
              ))}
            </select>
          </label>

          {items.length === 0 ? (
            <p style={{ opacity: 0.8 }}>
              住所がありません。先に「住所」ページで住所を登録してください。
            </p>
          ) : null}

          <button
            onClick={() => void onOrder()}
            disabled={isSubmitting || items.length === 0}
          >
            注文を確定する
          </button>
        </div>
      ) : (
        <p>loading...</p>
      )}
    </div>
  );
}
