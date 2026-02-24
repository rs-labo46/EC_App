import { Link } from "react-router-dom";
import { ApiError, listProducts, type ProductList } from "../api";
import { useEffect, useState } from "react";
import { ui } from "../ui/styles";

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
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>Products</h2>
        <p style={ui.subtitle}>気になる商品を選んで詳細を確認できます。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}

        {!data ? (
          <p style={ui.hint}>loading...</p>
        ) : (
          <div style={ui.grid}>
            {data.items.map((p) => (
              <div key={p.id} style={ui.productCard}>
                <p style={ui.productName}>
                  <Link to={`/products/${p.id}`}>{p.name}</Link>
                </p>
                <div style={ui.price}>¥{p.price}</div>
                <div style={ui.hint}>詳細で説明と在庫を確認できます</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
