import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  ApiError,
  adminCreateProduct,
  getMe,
  type AdminCreateProductRequest,
  type ApiSuccessShape,
  type Product,
} from "../api";
import { useAuth } from "../auth";

function toNumber(raw: string): number | null {
  if (raw.trim() === "") return null;
  const n: number = Number(raw);
  if (!Number.isFinite(n)) return null;
  return n;
}

function isSuccess(v: unknown): v is ApiSuccessShape {
  return (
    typeof v === "object" &&
    v !== null &&
    "message" in v &&
    typeof (v as { message: unknown }).message === "string"
  );
}

export default function AdminProductCreatePage() {
  const nav = useNavigate();
  const { accessToken } = useAuth();

  const [isChecking, setIsChecking] = useState<boolean>(true);
  const [isAdmin, setIsAdmin] = useState<boolean>(false);

  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [priceRaw, setPriceRaw] = useState<string>("0");
  const [stockRaw, setStockRaw] = useState<string>("0");
  const [isActive, setIsActive] = useState<boolean>(true);

  const [error, setError] = useState<string>("");
  const [msg, setMsg] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const price: number | null = useMemo(() => toNumber(priceRaw), [priceRaw]);
  const stock: number | null = useMemo(() => toNumber(stockRaw), [stockRaw]);

  // adminガード：/meでrole確認
  useEffect(() => {
    let alive = true;

    async function check(): Promise<void> {
      setIsChecking(true);
      setError("");

      if (!accessToken) {
        //未ログインならログインへ
        nav("/login?next=/admin/products/new");
        return;
      }

      try {
        const me = await getMe(accessToken);
        if (!alive) return;

        const ok: boolean = me.role === "ADMIN" && me.is_active;
        setIsAdmin(ok);

        if (!ok) {
          setError("管理者のみアクセスできます");
        }
      } catch (e: unknown) {
        if (!alive) return;
        setError(e instanceof ApiError ? e.message : "予期せぬエラー");
      } finally {
        if (!alive) return;
        setIsChecking(false);
      }
    }

    void check();
    return () => {
      alive = false;
    };
  }, [accessToken, nav]);

  const canSubmit: boolean = useMemo(() => {
    if (!isAdmin) return false;
    if (isSubmitting) return false;
    if (name.trim().length === 0) return false;
    if (price === null || price < 0) return false;
    if (stock === null || stock < 0) return false;
    return true;
  }, [isAdmin, isSubmitting, name, price, stock]);

  async function onSubmit(): Promise<void> {
    if (!accessToken) return;
    if (!canSubmit) return;

    setError("");
    setMsg("");
    setIsSubmitting(true);

    const req: AdminCreateProductRequest = {
      name: name.trim(),
      description: description.trim() ? description.trim() : undefined,
      price: price ?? 0,
      stock: stock ?? 0,
      is_active: isActive,
    };

    try {
      const res = await adminCreateProduct(accessToken, req);

      // res が {message} or Product のどっちでもOK
      if (isSuccess(res)) {
        setMsg(res.message);
      } else {
        const p = res as Product;
        setMsg(`created (id=${p.id})`);
      }

      // 作成後は一覧へ戻す（一覧がないなら /productsへでもOK）
      nav("/products");
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  if (isChecking) {
    return (
      <div style={{ display: "grid", gap: 12 }}>
        <h2>管理者の商品作成ページです</h2>
        <p>checking...</p>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div style={{ display: "grid", gap: 12 }}>
        <h2>管理者の商品作成ページです</h2>
        {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
        <button onClick={() => nav("/")}>トップへ戻る</button>
      </div>
    );
  }

  return (
    <div style={{ display: "grid", gap: 12, maxWidth: 520 }}>
      <h2>商品作成</h2>

      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}
      {msg ? <p style={{ color: "lime" }}>{msg}</p> : null}

      <label style={{ display: "grid", gap: 4 }}>
        <span>商品名</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={isSubmitting}
        />
      </label>

      <label style={{ display: "grid", gap: 4 }}>
        <span>商品説明</span>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={4}
          disabled={isSubmitting}
        />
      </label>

      <label style={{ display: "grid", gap: 4 }}>
        <span>価格</span>
        <input
          type="number"
          min={0}
          value={priceRaw}
          onChange={(e) => setPriceRaw(e.target.value)}
          disabled={isSubmitting}
        />
      </label>

      <label style={{ display: "grid", gap: 4 }}>
        <span>数量</span>
        <input
          type="number"
          min={0}
          value={stockRaw}
          onChange={(e) => setStockRaw(e.target.value)}
          disabled={isSubmitting}
        />
      </label>

      <label style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <input
          type="checkbox"
          checked={isActive}
          onChange={(e) => setIsActive(e.target.checked)}
          disabled={isSubmitting}
        />
        <span>公開する</span>
      </label>

      <div style={{ display: "flex", gap: 8 }}>
        <button onClick={() => void onSubmit()} disabled={!canSubmit}>
          作成
        </button>
        <button onClick={() => nav("/products")} disabled={isSubmitting}>
          戻る
        </button>
      </div>
    </div>
  );
}
