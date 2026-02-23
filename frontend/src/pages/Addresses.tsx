import { useEffect, useMemo, useState } from "react";
import {
  ApiError,
  createAddress,
  deleteAddress,
  listAddresses,
  setDefaultAddress,
  type Address,
  type AddressList,
} from "../api";
import { useAuth } from "../auth";

type NewAddressForm = {
  postal_code: string;
  prefecture: string;
  city: string;
  line1: string;
  line2: string;
  name: string;
  phone: string;
};

function isAddressList(value: unknown): value is AddressList {
  if (typeof value !== "object" || value === null) return false;
  if (!("items" in value)) return false;
  const items = (value as { items: unknown }).items;
  return Array.isArray(items);
}

function isAddressArray(value: unknown): value is Address[] {
  return Array.isArray(value);
}

export default function AddressesPage() {
  const { accessToken } = useAuth();

  const [data, setData] = useState<AddressList | null>(null);
  const [error, setError] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const [form, setForm] = useState<NewAddressForm>({
    postal_code: "",
    prefecture: "",
    city: "",
    line1: "",
    line2: "",
    name: "",
    phone: "",
  });

  // items が無くても落ちない
  const items: Address[] = useMemo(() => {
    if (!data) return [];
    return Array.isArray(data.items) ? data.items : [];
  }, [data]);

  function validateForm(f: NewAddressForm): string | null {
    if (!f.postal_code.trim()) return "郵便番号は必須です";
    if (!f.prefecture.trim()) return "都道府県は必須です";
    if (!f.city.trim()) return "市区町村は必須です";
    if (!f.line1.trim()) return "住所（line1）は必須です";
    if (!f.name.trim()) return "宛名（name）は必須です";
    return null;
  }

  async function load(): Promise<void> {
    if (!accessToken) return;

    setIsLoading(true);
    setError("");

    try {
      const res: unknown = await listAddresses(accessToken);
      if (isAddressList(res)) {
        setData(res);
        return;
      }
      if (isAddressArray(res)) {
        setData({ items: res });
        return;
      }

      // 期待外の形
      setData({ items: [] });
      setError(
        "住所一覧のレスポンス形式が想定と違います（items が見つかりません）",
      );
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
      setData({ items: [] });
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    // accessToken が変わったら取り直す
    void load();
  }, [accessToken]);

  async function onCreate(): Promise<void> {
    if (!accessToken) return;

    const msg = validateForm(form);
    if (msg) {
      setError(msg);
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      await createAddress(accessToken, {
        postal_code: form.postal_code.trim(),
        prefecture: form.prefecture.trim(),
        city: form.city.trim(),
        line1: form.line1.trim(),
        line2: form.line2.trim() ? form.line2.trim() : null,
        name: form.name.trim(),
        phone: form.phone.trim() ? form.phone.trim() : null,
      });
      // 登録後リロード
      await load();
      // 入力を軽くリセット
      setForm((prev) => ({ ...prev, line2: "" }));
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function onDelete(id: number): Promise<void> {
    if (!accessToken) return;

    setIsSubmitting(true);
    setError("");

    try {
      await deleteAddress(accessToken, id);
      await load();
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function onSetDefault(id: number): Promise<void> {
    if (!accessToken) return;

    setIsSubmitting(true);
    setError("");

    try {
      await setDefaultAddress(accessToken, id);
      await load();
    } catch (e: unknown) {
      setError(e instanceof ApiError ? e.message : "予期せぬエラー");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div style={{ display: "grid", gap: 14, maxWidth: 720 }}>
      <h2>住所</h2>

      {error ? <p style={{ color: "tomato" }}>{error}</p> : null}

      <div style={{ border: "1px solid #333", padding: 12 }}>
        <h3>住所登録</h3>
        <div style={{ display: "grid", gap: 6 }}>
          <input
            placeholder="5300001"
            value={form.postal_code}
            onChange={(e) => setForm({ ...form, postal_code: e.target.value })}
          />
          <input
            placeholder="大阪府"
            value={form.prefecture}
            onChange={(e) => setForm({ ...form, prefecture: e.target.value })}
          />
          <input
            placeholder="大阪市"
            value={form.city}
            onChange={(e) => setForm({ ...form, city: e.target.value })}
          />
          <input
            placeholder="ECサイト1-2-3"
            value={form.line1}
            onChange={(e) => setForm({ ...form, line1: e.target.value })}
          />
          <input
            placeholder="line2（任意）"
            value={form.line2}
            onChange={(e) => setForm({ ...form, line2: e.target.value })}
          />
          <input
            placeholder="EC 太郎"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <input
            placeholder="000-0000-0000（任意）"
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />

          <button onClick={() => void onCreate()} disabled={isSubmitting}>
            登録する
          </button>
        </div>
      </div>

      <div>
        <h3>登録した住所</h3>

        {isLoading ? <p>loading...</p> : null}

        {!isLoading && items.length === 0 ? (
          <p style={{ opacity: 0.8 }}>住所がありません</p>
        ) : (
          <ul>
            {items.map((a: Address) => (
              <li key={a.id} style={{ marginBottom: 10 }}>
                <div>
                  <b>
                    {a.name} {a.is_default ? "(default)" : ""}
                  </b>
                </div>
                <div>
                  {a.postal_code} {a.prefecture} {a.city} {a.line1}{" "}
                  {a.line2 ?? ""}
                  {a.phone}
                </div>
                <div style={{ display: "flex", gap: 8 }}>
                  <button
                    onClick={() => void onSetDefault(a.id)}
                    disabled={isSubmitting}
                  >
                    デフォルト
                  </button>
                  <button
                    onClick={() => void onDelete(a.id)}
                    disabled={isSubmitting}
                  >
                    削除
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
