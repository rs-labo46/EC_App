//APIアクセスをここに集約
export type ApiErrorShape = { error: string };
export type ApiSuccessShape = { message: string };

//バックエンドの User schema
export type User = {
  id: number;
  email: string;
  role: "USER" | "ADMIN";
  token_version: number;
  is_active: boolean;
};

//アクセストークンのレスポンス
export type JwtAccessToken = {
  access_token: string;
  expires_in: number;
  token_version: number;
};

export type AuthLoginResponse = {
  user: User;
  token: JwtAccessToken;
};

export type Product = {
  id: number;
  name: string;
  description?: string;
  price: number;
  stock: number;
  is_active: boolean;
  created_at: string;
};

export type ProductList = {
  items: Product[];
  total: number;
};

export type CartItem = {
  id: number;
  product_id: number;
  name: string;
  price: number;
  quantity: number;
};

export type CartResponse = {
  items: CartItem[];
  total: number;
};

export type Address = {
  id: number;
  postal_code: string;
  prefecture: string;
  city: string;
  line1: string;
  line2?: string | null;
  name: string;
  phone?: string | null;
  is_default: boolean;
};

export type AddressList = {
  items: Address[];
};

export type OrderItem = {
  product_id: number;
  name: string;
  price: number;
  quantity: number;
};

export type Order = {
  id: number;
  user_id: number;
  status: "PENDING" | "PAID" | "SHIPPED" | "CANCELED";
  total_price: number;
  created_at: string;
  items: OrderItem[];
};

export class ApiError extends Error {
  public readonly status: number;
  public readonly body?: ApiErrorShape;

  constructor(message: string, status: number, body?: ApiErrorShape) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

const API_BASE_URL: string =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

// Cookieから値を取り出す（CSRF Double Submit用）

function getCookieValue(name: string): string | null {
  const raw: string = document.cookie;
  if (!raw) return null;

  const parts: string[] = raw.split(";").map((p) => p.trim());
  for (const part of parts) {
    const eqIndex: number = part.indexOf("=");
    if (eqIndex < 0) continue;
    const k: string = decodeURIComponent(part.slice(0, eqIndex));
    if (k !== name) continue;
    const v: string = decodeURIComponent(part.slice(eqIndex + 1));
    return v;
  }
  return null;
}

type RequestOptions = {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  accessToken?: string | null;
  //body を送るとき
  jsonBody?: unknown;
  //CSRFが必要なAPI（refresh/logoutなど）だけtrueにする
  requireCsrf?: boolean;
  //注文確定などの idempotency が必要な時だけ付ける
  idempotencyKey?: string;
  //401のときrefreshを試すか
  retryOnUnauthorized?: boolean;
};

async function parseJsonSafe(res: Response): Promise<unknown | null> {
  const contentType: string | null = res.headers.get("content-type");
  if (!contentType || !contentType.includes("application/json")) {
    return null;
  }
  try {
    return await res.json();
  } catch {
    return null;
  }
}

async function request<T>(opts: RequestOptions): Promise<T> {
  const url: string = `${API_BASE_URL}${opts.path}`;

  const headers: Record<string, string> = {
    Accept: "application/json",
  };

  //JSON bodyがあるならContent-Typeを付ける
  if (opts.jsonBody !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  //bearerAuth（JWT）
  if (opts.accessToken) {
    headers["Authorization"] = `Bearer ${opts.accessToken}`;
  }

  //CSRF Double Submit
  if (opts.requireCsrf) {
    const csrf: string | null = getCookieValue("csrf_token");
    if (csrf) {
      headers["X-CSRF-Token"] = csrf;
    } else {
      throw new ApiError("csrf_token cookie not found", 0);
    }
  }

  // Idempotency（注文確定）
  if (opts.idempotencyKey) {
    headers["X-Idempotency-Key"] = opts.idempotencyKey;
  }

  const res: Response = await fetch(url, {
    method: opts.method,
    headers,
    body:
      opts.jsonBody !== undefined ? JSON.stringify(opts.jsonBody) : undefined,
    credentials: "include", //refresh cookieを送る
  });

  //401のとき、1回だけ refresh を試してから同じリクエストをやり直す
  if (res.status === 401 && (opts.retryOnUnauthorized ?? true)) {
    const refreshed: boolean = await tryRefresh();
    if (refreshed) {
      throw new ApiError(
        "unauthorized (token refreshed; retry with new access token)",
        401,
      );
    }
  }

  const data: unknown | null = await parseJsonSafe(res);

  if (!res.ok) {
    const body: ApiErrorShape | undefined =
      data && typeof data === "object" && data !== null && "error" in data
        ? (data as ApiErrorShape)
        : undefined;

    const message: string = body?.error ?? `request failed (${res.status})`;
    throw new ApiError(message, res.status, body);
  }

  //成功時はJSONを期待
  return data as T;
}

// refreshを試す（CSRF必須）
export async function authRefresh(): Promise<JwtAccessToken> {
  return request<JwtAccessToken>({
    method: "POST",
    path: "/auth/refresh",
    requireCsrf: true,
    retryOnUnauthorized: false,
  });
}

async function tryRefresh(): Promise<boolean> {
  try {
    await authRefresh();
    return true;
  } catch {
    return false;
  }
}
export type AuthRegisterResponse = { message: string };
export async function authRegister(
  email: string,
  password: string,
): Promise<AuthRegisterResponse> {
  return request<AuthRegisterResponse>({
    method: "POST",
    path: "/auth/register",
    jsonBody: { email, password },
    retryOnUnauthorized: false,
  });
}

//用途別API
export async function authLogin(
  email: string,
  password: string,
): Promise<AuthLoginResponse> {
  return request<AuthLoginResponse>({
    method: "POST",
    path: "/auth/login",
    jsonBody: { email, password },
    retryOnUnauthorized: false,
  });
}

export async function authLogout(
  accessToken: string,
): Promise<ApiSuccessShape> {
  return request<ApiSuccessShape>({
    method: "POST",
    path: "/auth/logout",
    accessToken,
    requireCsrf: true,
    retryOnUnauthorized: false,
  });
}

export async function getMe(accessToken: string): Promise<User> {
  return request<User>({
    method: "GET",
    path: "/me",
    accessToken,
  });
}

export async function listProducts(params: {
  page?: number;
  limit?: number;
  q?: string;
  min_price?: number;
  max_price?: number;
  sort?: "new" | "price_asc" | "price_desc";
}): Promise<ProductList> {
  const sp: URLSearchParams = new URLSearchParams();
  if (params.page) sp.set("page", String(params.page));
  if (params.limit) sp.set("limit", String(params.limit));
  if (params.q) sp.set("q", params.q);
  if (params.min_price !== undefined)
    sp.set("min_price", String(params.min_price));
  if (params.max_price !== undefined)
    sp.set("max_price", String(params.max_price));
  if (params.sort) sp.set("sort", params.sort);

  const query: string = sp.toString();
  return request<ProductList>({
    method: "GET",
    path: `/products${query ? `?${query}` : ""}`,
  });
}

export async function getProduct(id: number): Promise<Product> {
  return request<Product>({
    method: "GET",
    path: `/products/${id}`,
  });
}

export async function getCart(accessToken: string): Promise<CartResponse> {
  return request<CartResponse>({
    method: "GET",
    path: "/cart",
    accessToken,
  });
}

export async function addToCart(
  accessToken: string,
  productId: number,
  quantity: number,
): Promise<CartResponse> {
  return request<CartResponse>({
    method: "POST",
    path: "/cart",
    accessToken,
    jsonBody: { product_id: productId, quantity },
  });
}

export async function updateCartItem(
  accessToken: string,
  cartItemId: number,
  quantity: number,
): Promise<CartResponse> {
  return request<CartResponse>({
    method: "PATCH",
    path: `/cart/${cartItemId}`,
    accessToken,
    jsonBody: { quantity },
  });
}

export async function deleteCartItem(
  accessToken: string,
  cartItemId: number,
): Promise<CartResponse> {
  return request<CartResponse>({
    method: "DELETE",
    path: `/cart/${cartItemId}`,
    accessToken,
  });
}

export async function listAddresses(accessToken: string): Promise<AddressList> {
  return request<AddressList>({
    method: "GET",
    path: "/addresses",
    accessToken,
  });
}

export async function createAddress(
  accessToken: string,
  req: Omit<Address, "id" | "is_default">,
): Promise<Address> {
  return request<Address>({
    method: "POST",
    path: "/addresses",
    accessToken,
    jsonBody: req,
  });
}

export async function updateAddress(
  accessToken: string,
  id: number,
  req: Partial<Omit<Address, "id" | "is_default">>,
): Promise<Address> {
  return request<Address>({
    method: "PUT",
    path: `/addresses/${id}`,
    accessToken,
    jsonBody: req,
  });
}

export async function deleteAddress(
  accessToken: string,
  id: number,
): Promise<ApiSuccessShape> {
  return request<ApiSuccessShape>({
    method: "DELETE",
    path: `/addresses/${id}`,
    accessToken,
  });
}

export async function setDefaultAddress(
  accessToken: string,
  id: number,
): Promise<ApiSuccessShape> {
  return request<ApiSuccessShape>({
    method: "POST",
    path: `/addresses/${id}/default`,
    accessToken,
  });
}

//  注文確定（X-Idempotency-Key必須）

export async function createOrder(
  accessToken: string,
  addressId: number,
  idempotencyKey: string,
): Promise<Order> {
  return request<Order>({
    method: "POST",
    path: "/orders",
    accessToken,
    idempotencyKey,
    jsonBody: { address_id: addressId },
  });
}

export async function listOrders(accessToken: string): Promise<Order[]> {
  return request<Order[]>({
    method: "GET",
    path: "/orders",
    accessToken,
  });
}

export async function getOrder(
  accessToken: string,
  id: number,
): Promise<Order> {
  return request<Order>({
    method: "GET",
    path: `/orders/${id}`,
    accessToken,
  });
}

// ブラウザ標準のUUID生成
export function generateIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `idemp_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}
