import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ApiError, addToCart, getProduct, type Product } from "../api";
import { useAuth } from "../auth";

function clamp(n: number, min: number, max: number): number {
  if (n < min) return min;
  if (n > max) return max;
  return n;
}

function parseQty(raw: string): number | null {
  if (raw.trim() === "") return null;
  const n: number = Number(raw);
  if (!Number.isFinite(n)) return null;
  return n;
}

export default function ProductDetailPage() {
  const { id } = useParams();
  const nav = useNavigate();

  const productId: number = Number(id);
  const isValidProductId: boolean = Number.isFinite(productId) && productId > 0;

  const { accessToken } = useAuth();

  const [product, setProduct] = useState<Product | null>(null);
  const [qty, setQty] = useState<number>(1);
  const [msg, setMsg] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  // maxQtyはuseMemoで保存
  const maxQty: number = useMemo(() => {
    if (!product) return 1;
    return Math.max(1, product.stock);
  }, [product]);

  // プルダウン
  const qtyOptions: number[] = useMemo(() => {
    if (!product) return [1];
    const n: number = Math.max(1, product.stock);
    return Array.from({ length: n }, (_, i) => i + 1);
  }, [product]);

  useEffect(() => {
    let alive = true;

    async function load(): Promise<void> {
      if (!isValidProductId) {
        if (alive) {
          setProduct(null);
          setError("無効なIDです");
        }
        return;
      }

      try {
        setError("");
        const p = await getProduct(productId);
        if (!alive) return;

        setProduct(p);
        const stockMax: number = Math.max(1, p.stock);
        setQty((prev) => clamp(Number.isFinite(prev) ? prev : 1, 1, stockMax));
      } catch (e: unknown) {
        if (!alive) return;
        setProduct(null);
        setError(e instanceof ApiError ? e.message : "予期せぬerror");
      }
    }

    void load();
    return () => {
      alive = false;
    };
  }, [productId, isValidProductId]);

  async function onAdd(): Promise<void> {
    setMsg("");
    setError("");

    if (!accessToken) {
      setError("カートに追加するにはログインが必要です");
      return;
    }
    if (!product) return;

    if (product.stock <= 0) {
      setError("在庫切れです");
      return;
    }

    const safeQty: number = clamp(qty, 1, product.stock);

    setIsSubmitting(true);
    try {
      await addToCart(accessToken, productId, safeQty);
      setMsg("カートに追加しました");
      nav("/cart");
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬerror");
    } finally {
      setIsSubmitting(false);
    }
  }

  if (!isValidProductId) return <p>無効なIDです</p>;

  const canAdd: boolean =
    !!accessToken && !!product && product.stock > 0 && !isSubmitting;

  return (
    <div>
      <h2>商品詳細</h2>

      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {msg ? <p style={{ color: "lime" }}>{msg}</p> : null}

      {!product ? (
        <p>loading...</p>
      ) : (
        <div style={{ display: "grid", gap: 8, maxWidth: 520 }}>
          <div>
            <b>{product.name}</b>
          </div>
          <div>¥{product.price}</div>
          <div style={{ opacity: 0.9 }}>{product.description}</div>

          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <select
              value={qty}
              onChange={(e) => {
                const parsed: number | null = parseQty(e.target.value);
                if (parsed === null) return;
                setQty(clamp(parsed, 1, maxQty));
              }}
              disabled={product.stock <= 0 || isSubmitting}
              style={{ width: 80 }}
            >
              {qtyOptions.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>

            <button onClick={() => void onAdd()} disabled={!canAdd}>
              カートに追加
            </button>

            <span style={{ opacity: 0.8, fontSize: 12 }}>
              {!accessToken
                ? "ログイン必須"
                : product.stock <= 0
                  ? "在庫切れ"
                  : ""}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
