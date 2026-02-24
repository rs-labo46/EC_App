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
import { ui } from "../ui/styles";

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
    <div style={ui.page}>
      <div style={ui.header}>
        <h2 style={ui.title}>住所</h2>
        <p style={ui.subtitle}>配送先を登録・デフォルト設定・削除できます。</p>
      </div>

      <div style={ui.cardWide}>
        {error ? <p style={ui.msgErr}>{error}</p> : null}

        <div style={{ display: "grid", gap: 12 }}>
          <div style={ui.productCard}>
            <div style={{ ...ui.productName, marginBottom: 8 }}>住所登録</div>

            <div style={{ display: "grid", gap: 10, maxWidth: 520 }}>
              <input
                placeholder="5300001"
                value={form.postal_code}
                onChange={(e) =>
                  setForm({ ...form, postal_code: e.target.value })
                }
                style={ui.input}
              />
              <input
                placeholder="大阪府"
                value={form.prefecture}
                onChange={(e) =>
                  setForm({ ...form, prefecture: e.target.value })
                }
                style={ui.input}
              />
              <input
                placeholder="大阪市"
                value={form.city}
                onChange={(e) => setForm({ ...form, city: e.target.value })}
                style={ui.input}
              />
              <input
                placeholder="ECサイト1-2-3"
                value={form.line1}
                onChange={(e) => setForm({ ...form, line1: e.target.value })}
                style={ui.input}
              />
              <input
                placeholder="line2（任意）"
                value={form.line2}
                onChange={(e) => setForm({ ...form, line2: e.target.value })}
                style={ui.input}
              />
              <input
                placeholder="EC 太郎"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                style={ui.input}
              />
              <input
                placeholder="000-0000-0000（任意）"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
                style={ui.input}
              />

              <button
                onClick={() => void onCreate()}
                disabled={isSubmitting}
                style={{
                  ...ui.buttonPrimary,
                  ...(isSubmitting ? ui.buttonPrimaryDisabled : null),
                }}
              >
                登録する
              </button>
            </div>
          </div>

          <div style={ui.productCard}>
            <div style={{ ...ui.productName, marginBottom: 8 }}>
              登録した住所
            </div>

            {isLoading ? <p style={ui.hint}>loading...</p> : null}

            {!isLoading && items.length === 0 ? (
              <p style={ui.hint}>住所がありません</p>
            ) : (
              <div style={{ display: "grid", gap: 10 }}>
                {items.map((a: Address) => (
                  <div
                    key={a.id}
                    style={{
                      padding: "12px 12px",
                      borderRadius: 14,
                      border: "1px solid rgba(255,255,255,0.10)",
                      background: "rgba(0,0,0,0.18)",
                    }}
                  >
                    <div style={{ fontWeight: 900 }}>
                      {a.name} {a.is_default ? "(default)" : ""}
                    </div>

                    <div style={{ ...ui.hint, marginTop: 6, lineHeight: 1.6 }}>
                      {a.postal_code} {a.prefecture} {a.city} {a.line1}{" "}
                      {a.line2 ?? ""} {a.phone}
                    </div>

                    <div
                      style={{
                        display: "flex",
                        gap: 10,
                        marginTop: 10,
                        flexWrap: "wrap",
                      }}
                    >
                      <button
                        onClick={() => void onSetDefault(a.id)}
                        disabled={isSubmitting}
                        style={{
                          ...ui.buttonPrimary,
                          ...(isSubmitting ? ui.buttonPrimaryDisabled : null),
                        }}
                      >
                        デフォルト
                      </button>
                      <button
                        onClick={() => void onDelete(a.id)}
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
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
