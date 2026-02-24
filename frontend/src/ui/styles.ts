// src/ui/styles.ts
// 目的: 画面ごとにバラバラな見た目にならないよう、共通スタイルをここに集約する

import type { CSSProperties } from "react";

// テーマ（色・角丸・影などの共通ルール）
const theme = {
  pageBg: "#0b1020",
  cardBg: "rgba(255, 255, 255, 0.06)",
  cardBorder: "rgba(255, 255, 255, 0.12)",
  text: "rgba(255, 255, 255, 0.92)",
  subText: "rgba(255, 255, 255, 0.72)",
  danger: "#ff5a5f",
  success: "#3ddc97",
  primary: "#7c5cff",
  primaryHover: "#6b4cff",
  radius: 16,
  shadow: "0 10px 30px rgba(0,0,0,0.35)",
  maxWidthSm: 420,
  maxWidthMd: 720,
} as const;

// 共通スタイル集（CSSPropertiesで型安全に）
export const ui = {
  // 画面全体の土台（背景・中央寄せ・余白）
  page: {
    minHeight: "100vh",
    background: `radial-gradient(1200px 600px at 20% 0%, rgba(124,92,255,0.22), transparent 55%),
                 radial-gradient(900px 500px at 80% 10%, rgba(61,220,151,0.16), transparent 55%),
                 ${theme.pageBg}`,
    color: theme.text,
    padding: "48px 16px",
  } satisfies CSSProperties,

  // ヘッダー（ページタイトル周り）
  header: {
    maxWidth: theme.maxWidthMd,
    margin: "0 auto 16px",
  } satisfies CSSProperties,

  title: {
    fontSize: 26,
    letterSpacing: 0.2,
    margin: 0,
  } satisfies CSSProperties,

  subtitle: {
    marginTop: 6,
    marginBottom: 0,
    color: theme.subText,
    fontSize: 13,
    lineHeight: 1.6,
  } satisfies CSSProperties,

  // カード（フォームや詳細を入れる箱）
  card: {
    maxWidth: theme.maxWidthSm,
    margin: "0 auto",
    background: theme.cardBg,
    border: `1px solid ${theme.cardBorder}`,
    borderRadius: theme.radius,
    boxShadow: theme.shadow,
    padding: 18,
    backdropFilter: "blur(10px)",
  } satisfies CSSProperties,

  // 広めカード（一覧など）
  cardWide: {
    maxWidth: theme.maxWidthMd,
    margin: "0 auto",
    background: theme.cardBg,
    border: `1px solid ${theme.cardBorder}`,
    borderRadius: theme.radius,
    boxShadow: theme.shadow,
    padding: 18,
    backdropFilter: "blur(10px)",
  } satisfies CSSProperties,

  // フォーム
  form: {
    display: "grid",
    gap: 12,
    marginTop: 12,
  } satisfies CSSProperties,

  label: {
    display: "grid",
    gap: 6,
    fontSize: 13,
    color: theme.subText,
  } satisfies CSSProperties,

  input: {
    width: "100%",
    padding: "10px 12px",
    borderRadius: 12,
    border: `1px solid ${theme.cardBorder}`,
    background: "rgba(0,0,0,0.25)",
    color: theme.text,
    outline: "none",
  } satisfies CSSProperties,

  // selectもinputと同じ見た目にする
  select: {
    padding: "10px 12px",
    borderRadius: 12,
    border: `1px solid ${theme.cardBorder}`,
    background: "rgba(0,0,0,0.25)",
    color: theme.text,
    outline: "none",
  } satisfies CSSProperties,

  // ボタン（主役）
  buttonPrimary: {
    padding: "10px 14px",
    borderRadius: 12,
    border: "none",
    background: theme.primary,
    color: "white",
    fontWeight: 700,
    cursor: "pointer",
  } satisfies CSSProperties,

  buttonPrimaryDisabled: {
    opacity: 0.55,
    cursor: "not-allowed",
  } satisfies CSSProperties,

  // メッセージ
  msgOk: {
    marginTop: 12,
    color: theme.success,
    fontSize: 13,
  } satisfies CSSProperties,

  msgErr: {
    marginTop: 12,
    color: theme.danger,
    fontSize: 13,
  } satisfies CSSProperties,

  // リンク行（小さめ）
  footnote: {
    marginTop: 12,
    fontSize: 12,
    color: theme.subText,
  } satisfies CSSProperties,

  // 商品一覧：グリッドカード
  grid: {
    display: "grid",
    gap: 12,
    gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
    marginTop: 12,
  } satisfies CSSProperties,

  productCard: {
    background: "rgba(0,0,0,0.22)",
    border: `1px solid ${theme.cardBorder}`,
    borderRadius: 14,
    padding: 14,
  } satisfies CSSProperties,

  productName: {
    fontWeight: 800,
    margin: 0,
    fontSize: 14,
  } satisfies CSSProperties,

  price: {
    marginTop: 8,
    color: theme.text,
    fontWeight: 800,
  } satisfies CSSProperties,

  metaRow: {
    display: "flex",
    gap: 10,
    alignItems: "center",
    flexWrap: "wrap",
    marginTop: 12,
  } satisfies CSSProperties,

  hint: {
    color: theme.subText,
    fontSize: 12,
  } satisfies CSSProperties,

  navWrap: {
    position: "sticky",
    top: 0,
    zIndex: 50,
    background: "rgba(11,16,32,0.65)",
    backdropFilter: "blur(10px)",
    borderBottom: "1px solid rgba(255,255,255,0.10)",
  } satisfies CSSProperties,

  navInner: {
    width: "min(1100px, 100%)",
    margin: "0 auto",
    padding: "10px 16px",
    display: "flex",
    gap: 12,
    alignItems: "center",
  } satisfies CSSProperties,

  navRight: {
    marginLeft: "auto",
    display: "flex",
    gap: 12,
    alignItems: "center",
  } satisfies CSSProperties,

  navLink: {
    color: "rgba(255,255,255,0.85)",
    textDecoration: "none",
    fontWeight: 700,
  } satisfies CSSProperties,
} as const;
