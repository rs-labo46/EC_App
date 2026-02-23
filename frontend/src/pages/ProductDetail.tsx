import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { ApiError, addToCart, getProduct, type Product } from "../api";
import { useAuth } from "../auth";

export default function ProductDetailPage() {
  const { id } = useParams();
  const productId: number = Number(id);

  const { accessToken } = useAuth();

  const [product, setProduct] = useState<Product | null>(null);
  const [qty, setQty] = useState<number>(1);
  const [msg, setMsg] = useState<string>("");
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let alive = true;
    async function load(): Promise<void> {
      try {
        const p = await getProduct(productId);
        if (!alive) return;
        setProduct(p);
      } catch (e: unknown) {
        if (!alive) return;
        setError(e instanceof ApiError ? e.message : "予期せぬerror");
      }
    }
    if (!Number.isFinite(productId)) return;
    void load();
    return () => {
      alive = false;
    };
  }, [productId]);

  async function onAdd(): Promise<void> {
    setMsg("");
    setError("");
    if (!accessToken) {
      setError("カートに追加するにはログインが必要です");
      return;
    }
    try {
      await addToCart(accessToken, productId, qty);
      setMsg("カートに追加");
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬerror");
    }
  }

  if (!Number.isFinite(productId)) return <p>無効なIDです</p>;

  return (
    <div>
      <h2>Product Detail</h2>
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
          <div>stock: {product.stock}</div>
          <div style={{ opacity: 0.9 }}>{product.description}</div>

          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <input
              type="number"
              min={1}
              value={qty}
              onChange={(e) => setQty(Number(e.target.value))}
              style={{ width: 80 }}
            />
            <button onClick={() => void onAdd()}>Add to Cart</button>
            <span style={{ opacity: 0.8, fontSize: 12 }}>ログイン必須</span>
          </div>
        </div>
      )}
    </div>
  );
}
