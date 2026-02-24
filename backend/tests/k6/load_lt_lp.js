import http from "k6/http";
import { check, sleep } from "k6";
import exec from "k6/execution";

// =====================
// Env
// =====================
const BASE_URL = __ENV.BASE_URL || "http://127.0.0.1:8080";

const USER_EMAIL_PREFIX = __ENV.USER_EMAIL_PREFIX || "user+";
const USER_EMAIL_DOMAIN = __ENV.USER_EMAIL_DOMAIN || "test.com";
const USER_PASSWORD = __ENV.USER_PASSWORD || "CorrectPW123!";

const USER_POOL_SIZE = parseInt(__ENV.USER_POOL_SIZE || "200", 10);

const FALLBACK_PRODUCT_ID = parseInt(__ENV.FALLBACK_PRODUCT_ID || "1", 10);

// orders handler は body.address_id を見る
const ADDRESS_ID = parseInt(__ENV.ADDRESS_ID || "1", 10);

// Cookie名 / Header名（ハンドラー定義に合わせて固定）
const COOKIE_REFRESH = "refresh_token";
const COOKIE_CSRF = "csrf_token";
const HEADER_CSRF = "X-CSRF-Token";
const HEADER_IDEMPOTENCY = "X-Idempotency-Key";

// sleep
const SLEEP_MIN_MS = parseInt(__ENV.SLEEP_MIN_MS || "50", 10);
const SLEEP_MAX_MS = parseInt(__ENV.SLEEP_MAX_MS || "200", 10);

// =====================
// LT1〜LT6 比率
// =====================
const MIX = [
  { name: "LT1_products_list", weight: 50 },
  { name: "LT2_products_detail", weight: 20 },
  { name: "LT3_cart_get", weight: 10 },
  { name: "LT4_cart_add", weight: 10 },
  { name: "LT5_orders_place", weight: 5 },
  { name: "LT6_auth_refresh", weight: 5 },
];

// =====================
// LP1〜LP5：TEST_PROFILE で切替
// =====================
function makeOptions() {
  const profile = (__ENV.TEST_PROFILE || "smoke").toLowerCase();

  const targetRps = Number(__ENV.TARGET_RPS || "16.7"); // 1000/min ≒ 16.7 rps
  const spikeRps = Number(__ENV.SPIKE_RPS || "50");
  const limitRps = Number(__ENV.LIMIT_RPS || "120");

  const smokeDuration = __ENV.SMOKE_DURATION || "5m";
  const targetDuration = __ENV.TARGET_DURATION || "30m";
  const enduranceDuration = __ENV.ENDURANCE_DURATION || "2h";
  const spikeHold = __ENV.SPIKE_HOLD || "3m";
  const limitHold = __ENV.LIMIT_HOLD || "2m";

  const baseScenario = (ratePerSec, duration, maxVUs) => ({
    executor: "constant-arrival-rate",
    rate: Math.ceil(ratePerSec),
    timeUnit: "1s",
    duration,
    preAllocatedVUs: Math.max(20, Math.floor(maxVUs / 2)),
    maxVUs,
  });

  const thresholds = {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<400"],

    "http_req_duration{name:LT1_products_list}": ["p(95)<200"],
    "http_req_duration{name:LT2_products_detail}": ["p(95)<300"],
    "http_req_duration{name:LT3_cart_get}": ["p(95)<300"],
    "http_req_duration{name:LT4_cart_add}": ["p(95)<300"],
    "http_req_duration{name:LT5_orders_place}": ["p(95)<300"],
    "http_req_duration{name:LT6_auth_refresh}": ["p(95)<300"],
  };

  if (profile === "smoke") {
    const rps = Number(__ENV.SMOKE_RPS || "2");
    return {
      thresholds,
      scenarios: { load: baseScenario(rps, smokeDuration, 50) },
    };
  }

  if (profile === "target") {
    return {
      thresholds,
      scenarios: { load: baseScenario(targetRps, targetDuration, 200) },
    };
  }

  if (profile === "endurance") {
    return {
      thresholds,
      scenarios: { load: baseScenario(targetRps, enduranceDuration, 200) },
    };
  }

  if (profile === "spike") {
    return {
      thresholds,
      scenarios: {
        load: {
          executor: "ramping-arrival-rate",
          startRate: Math.ceil(targetRps),
          timeUnit: "1s",
          preAllocatedVUs: 100,
          maxVUs: 400,
          stages: [
            { target: Math.ceil(targetRps), duration: "1m" },
            { target: Math.ceil(spikeRps), duration: "30s" },
            { target: Math.ceil(spikeRps), duration: spikeHold },
            { target: Math.ceil(targetRps), duration: "1m" },
          ],
        },
      },
    };
  }

  return {
    thresholds,
    scenarios: {
      load: {
        executor: "ramping-arrival-rate",
        startRate: 10,
        timeUnit: "1s",
        preAllocatedVUs: 200,
        maxVUs: 800,
        stages: [
          { target: 20, duration: "2m" },
          { target: 40, duration: "2m" },
          { target: 60, duration: "2m" },
          { target: 80, duration: "2m" },
          { target: Math.ceil(limitRps), duration: limitHold },
        ],
      },
    },
  };
}

export const options = makeOptions();

// =====================
// setup：商品ID候補を拾う（/products）
// =====================
export function setup() {
  const url = `${BASE_URL}/products?page=1&limit=20`;
  const res = http.get(url, { tags: { name: "LT1_products_list" } });

  if (res.status !== 200) {
    return { productIds: [FALLBACK_PRODUCT_ID] };
  }

  try {
    const body = res.json();
    const items = Array.isArray(body.items) ? body.items : [];
    const ids = items
      .map((p) => (typeof p.id === "number" ? p.id : null))
      .filter((x) => x !== null);

    return { productIds: ids.length > 0 ? ids : [FALLBACK_PRODUCT_ID] };
  } catch (_e) {
    return { productIds: [FALLBACK_PRODUCT_ID] };
  }
}

// =====================
// helpers
// =====================
function pickUserEmail() {
  const idx = ((exec.vu.idInTest - 1) % USER_POOL_SIZE) + 1;
  const padded = String(idx).padStart(4, "0");
  return `${USER_EMAIL_PREFIX}${padded}@${USER_EMAIL_DOMAIN}`;
}

function randomSleep() {
  const ms =
    SLEEP_MIN_MS +
    Math.floor(Math.random() * Math.max(1, SLEEP_MAX_MS - SLEEP_MIN_MS));
  sleep(ms / 1000);
}

function pickAction() {
  const total = MIX.reduce((s, x) => s + x.weight, 0);
  const r = Math.random() * total;
  let acc = 0;
  for (const x of MIX) {
    acc += x.weight;
    if (r <= acc) return x.name;
  }
  return MIX[MIX.length - 1].name;
}

//register helper
function tryRegister(email, password) {
  const payload = JSON.stringify({ email, password });

  // register は cookie 不要なので jarなしでOK（実装により不要なら付けても害は少ない）
  const res = http.post(`${BASE_URL}/auth/register`, payload, {
    headers: { "Content-Type": "application/json" },
    tags: { name: "AUTH_register" },
  });

  // 実装差を吸収：201/200/204 あたりを許容（409は既存ユーザー）
  check(res, {
    "register 200/201/204/409": (r) =>
      r.status === 200 ||
      r.status === 201 ||
      r.status === 204 ||
      r.status === 409,
  });

  return res.status;
}

// =====================
// login：/auth/login
// - access token は JSON の token.access_token
// - cookie: refresh_token / csrf_token
// =====================
function loginAndGetContext() {
  const jar = http.cookieJar();

  const email = pickUserEmail();
  const payload = JSON.stringify({ email, password: USER_PASSWORD });

  // 1回目 login
  let res = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: { "Content-Type": "application/json" },
    jar,
    tags: { name: "AUTH_login" },
  });

  // login が落ちたら、1回だけ register して再login
  if (res.status !== 200) {
    tryRegister(email, USER_PASSWORD);

    res = http.post(`${BASE_URL}/auth/login`, payload, {
      headers: { "Content-Type": "application/json" },
      jar,
      tags: { name: "AUTH_login" },
    });
  }

  const okStatus = check(res, { "login 200": (r) => r.status === 200 });
  if (!okStatus) return { ok: false, jar, access: "", csrf: "" };

  // access token 取得（token.access_token 前提）
  let access = "";
  try {
    const b = res.json();
    if (b && b.token && typeof b.token.access_token === "string") {
      access = b.token.access_token;
    }
  } catch (_e) {}

  // csrf cookie 取得
  let csrf = "";
  const cookies = jar.cookiesForURL(BASE_URL);
  if (cookies && cookies[COOKIE_CSRF] && cookies[COOKIE_CSRF].length > 0) {
    csrf = cookies[COOKIE_CSRF][0].value;
  }

  // “200だけど中身空” を弾く（ここが超重要：後続の401ノイズを防ぐ）
  const okPayload = access !== "" && csrf !== "";
  check({ access, csrf }, { "login has access+csrf": () => okPayload });

  if (!okPayload) return { ok: false, jar, access: "", csrf: "" };

  return { ok: true, jar, access, csrf };
}

// =====================
// LT actions
// =====================
function lt1ProductsList() {
  const res = http.get(`${BASE_URL}/products?page=1&limit=20`, {
    tags: { name: "LT1_products_list" },
  });
  check(res, { "LT1 200": (r) => r.status === 200 });
}

function lt2ProductsDetail(productId) {
  const res = http.get(`${BASE_URL}/products/${productId}`, {
    tags: { name: "LT2_products_detail" },
  });
  check(res, { "LT2 200/404": (r) => r.status === 200 || r.status === 404 });
}

function lt3CartGet(auth) {
  const res = http.get(`${BASE_URL}/cart`, {
    headers: { Authorization: `Bearer ${auth.access}` },
    jar: auth.jar,
    tags: { name: "LT3_cart_get" },
  });
  check(res, { "LT3 200": (r) => r.status === 200 });
}

function lt4CartAdd(auth, productId) {
  // CartHandler: { product_id, quantity }
  const payload = JSON.stringify({ product_id: productId, quantity: 1 });

  const res = http.post(`${BASE_URL}/cart`, payload, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${auth.access}`,
    },
    jar: auth.jar,
    tags: { name: "LT4_cart_add" },
  });

  // 在庫/仕様により 409 等があり得るので幅を持たせる
  check(res, {
    "LT4 200/400/409": (r) =>
      r.status === 200 || r.status === 400 || r.status === 409,
  });
}

function lt5OrdersPlace(auth, productId) {
  // 1) まずカートに入れる（空カート注文による 400 を消す）
  const addPayload = JSON.stringify({ product_id: productId, quantity: 1 });

  const addRes = http.post(`${BASE_URL}/cart`, addPayload, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${auth.access}`,
    },
    jar: auth.jar,
    tags: { name: "LT5_orders_place" },
  });

  // ここは「成功させたい」ので 200 を狙う（在庫がないなら productId 選びが必要）
  const addOk = check(addRes, { "LT5 cart_add 200": (r) => r.status === 200 });
  if (!addOk) {
    // 在庫切れ等のときは、この回は注文まで行かずに戻る（4xx増やさないため）
    // ※ログは必要なら追加
    return;
  }

  // 2) 注文
  const key = `${exec.vu.idInTest}-${Date.now()}-${Math.random()
    .toString(16)
    .slice(2)}`;

  const orderPayload = JSON.stringify({
    address_id: ADDRESS_ID,
    idempotency_key: key, // handlerは見ないが一応残してOK
  });

  const res = http.post(`${BASE_URL}/orders`, orderPayload, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${auth.access}`,
      [HEADER_IDEMPOTENCY]: key,
    },
    jar: auth.jar,
    tags: { name: "LT5_orders_place" },
  });

  const ok = check(res, { "LT5 200": (r) => r.status === 200 });
  if (!ok) logLt5Failure(res);
}
let __lt5LogCount = 0;

function logLt5Failure(res) {
  if (__lt5LogCount >= 3) return;
  __lt5LogCount++;

  console.log(
    `[LT5 fail] status=${res.status} body=${String(res.body).slice(0, 300)}`,
  );
}

function lt6AuthRefresh(auth) {
  const res = http.post(`${BASE_URL}/auth/refresh`, null, {
    headers: { [HEADER_CSRF]: auth.csrf },
    jar: auth.jar,
    tags: { name: "LT6_auth_refresh" },
  });

  const ok = check(res, { "LT6 200": (r) => r.status === 200 });
  if (!ok) {
    console.log(
      `[LT6 fail] status=${res.status} body=${String(res.body).slice(0, 300)}`,
    );
  }
}

// =====================
// default（arrival-rate 用：1回=1リクエスト）
// =====================
export default function (data) {
  if (!globalThis.__authCtx) {
    globalThis.__authCtx = loginAndGetContext();
  }

  const productIds =
    data && Array.isArray(data.productIds) && data.productIds.length > 0
      ? data.productIds
      : [FALLBACK_PRODUCT_ID];
  const productId = productIds[Math.floor(Math.random() * productIds.length)];

  const action = pickAction();

  // 認証必須のアクションは login 失敗ならスキップ
  if (
    !globalThis.__authCtx.ok &&
    (action === "LT3_cart_get" ||
      action === "LT4_cart_add" ||
      action === "LT5_orders_place" ||
      action === "LT6_auth_refresh")
  ) {
    sleep(0.1);
    return;
  }

  switch (action) {
    case "LT1_products_list":
      lt1ProductsList();
      break;
    case "LT2_products_detail":
      lt2ProductsDetail(productId);
      break;
    case "LT3_cart_get":
      lt3CartGet(globalThis.__authCtx);
      break;
    case "LT4_cart_add":
      lt4CartAdd(globalThis.__authCtx, productId);
      break;
    case "LT5_orders_place":
      lt5OrdersPlace(globalThis.__authCtx, productId);
      break;
    case "LT6_auth_refresh":
      lt6AuthRefresh(globalThis.__authCtx);
      break;
    default:
      lt1ProductsList();
      break;
  }

  randomSleep();
}
