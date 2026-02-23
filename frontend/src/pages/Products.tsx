import { Link } from "react-router-dom";
import { ApiError, listProducts, type ProductList } from "../api";
import { useEffect, useState } from "react";

export default function ProductsPage() {
  const [data, setData] = useState<ProductList | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let alive = true;

    async function load(): Promise<void> {
      try {
        const res = await listProducts({ page: 1, limit: 20 });
        if (!alive) return;
        setData(res);
      } catch (e: unknown) {
        if (!alive) return;
        setError(e instanceof ApiError ? e.message : "unexpected error");
      }
    }

    void load();
    return () => {
      alive = false;
    };
  }, []);

  return (
    <div>
      <h2>Products</h2>
      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {!data ? (
        <p>loading...</p>
      ) : (
        <ul>
          {data.items.map((p) => (
            <li key={p.id}>
              <Link to={`/products/${p.id}`}>{p.name}</Link> — ¥{p.price}
              （stock:{p.stock}）
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
