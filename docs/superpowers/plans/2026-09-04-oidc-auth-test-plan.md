# OpenMeter OIDC/Casdoor 认证接入测试方案

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为工作区中未提交的管理面认证接入（后端 OIDC 登录/回调 + 自有会话 JWT + 强制认证中间件 + 前端 SSO 入口/回调页）建立分层自动化测试与验收清单，并钉住"启用认证"这一行为面变化的影响范围。

**Architecture:** 后端 `openmeter/auth` 包实现 Authorization Code Flow（`/auth/oidc/login` → IdP → `/auth/oidc/callback`），验签 IdP 的 id_token 后签发自有 HS256 会话 JWT，经 URL fragment 交给前端 `/auth/callback` 页；中间件在 v1/v3 全组强制 Bearer 会话（豁免 `/api/v1/portal/` 与 `/api/swagger.json` 前缀），会话 organization 经 `SessionNamespaceDecoder` 映射为小写 namespace。测试分五层：后端单测（fake IdP）→ 后端集成（noop services 全栈路由）→ 前端组件单测（vitest+jsdom）→ 浏览器 E2E（Node 零依赖 fake IdP + Playwright）→ 真实 Casdoor 手工联调。

**Tech Stack:** Go 1.27 / chi / golang-jwt v5 / coreos go-oidc v3 / testify；React 19 / vitest 3 / @testing-library/react / Playwright；Node 零依赖 http + crypto（RS256）。

## Global Constraints

- 本方案**只新增测试与测试夹具，不修改生产代码**。测试暴露的缺陷记录到方案末尾风险表，另立修复任务。
- 本机无 nix：直接用 Homebrew go。模块已在缓存；如需下载依赖加前缀 `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn`（GOENV 里默认 GOPROXY=off）。
- auth 相关测试**不需要 PostgreSQL**（fake IdP + noop services 全覆盖）；`openmeter/server` 包测试统一带 `-tags=dynamic` 保平安。
- 浏览器验证一律用 front 自带 Playwright；不要依据 ZCode 内置浏览器的点击/截图结论下判断（历史经验：IAB 点击假象）。
- 命令一律从仓库根 `/Users/wuyongjun/trea/openmeter` 或 `front/` 执行；后端测试：`go test ./openmeter/auth/... ./app/config/... ./openmeter/namespace/namespacedriver/...`、`go test -tags=dynamic ./openmeter/server/...`；前端单测：`cd front && npx vitest run src/pages/auth`。
- 提交信息沿用仓库风格：后端 `test(auth): ...`，前端 `test(front): ...`。

---

## 一、被测对象与已核实事实

以下事实均已读源码核实（行号为当前工作区状态），是全部测试断言的依据。

### 后端（根模块，未提交）

| 行为 | 证据 |
|---|---|
| 路由 `GET /auth/oidc/login`、`GET /auth/oidc/callback` | `openmeter/auth/handler.go:142-147` |
| Login 生成 32 字节 hex state，写 cookie `om_oidc_state`（Path=/auth/oidc，10 分钟，HttpOnly，Lax）；前端 nonce `?state=` 原样写 cookie `om_oidc_login_state` | `handler.go:154-188` |
| Callback 校验 query state 与 cookie 常量时间比对，失败/缺 cookie/缺 code → 400；state cookie 单次使用（MaxAge=-1 清除） | `handler.go:204-225` |
| code 换 token 失败 → 502；响应无 id_token → 502；id_token 验签失败 → 401 | `handler.go:227-249` |
| 组织 claim 取 `organization` 优先、回退 `owner`；两者皆空时**仍签发 token**，fragment 不带 `tenant_id` | `handler.go:259-283` |
| 成功 302 到 `DashboardURL#token=<会话JWT>&tenant_id=<org>[&state=<nonce>]` | `handler.go:278-300` |
| 会话 JWT：HS256、iss=`openmeter`、claims sub/iat/exp/email/name/organization；无 aud/jti；Verify 强制 exp、锁 HS256、**无 clock leeway**；⚠️ `WithStrictDecoding` 实测（jwt/v5 v5.3.1）只校验 base64 填充，**不拒绝未知 claim 字段** | `openmeter/auth/session_token.go:85-129`，Task 1 实测钉住 |
| 中间件：仅认 `Authorization: Bearer`；豁免纯 HasPrefix；organization 为空 → 401；角色硬编码 admin；401 带 `WWW-Authenticate: Bearer realm="openmeter"` | `openmeter/auth/middleware.go:60-107` |
| server 级统一豁免前缀 `["/api/v1/portal/", "/api/swagger.json"]`；v3 组末尾追加、v1 组第一位前置 | `openmeter/server/server.go` diff |
| namespace：会话 OrgSlug 小写落地，无会话/空 OrgSlug **静默回退默认 namespace**（永不出错） | `openmeter/namespace/namespacedriver/session.go:25-31` |
| 配置：`auth.oidc.enabled` 默认 false；启用时校验 URL/凭据/secret；secret 走环境变量 `AUTH_OIDC_CLIENTSECRET`/`AUTH_TOKENSECRET`；`tokenExpiration` 默认 720h | `app/config/auth.go:40-114`，环境变量注入已被 `TestAuthConfigurationEnvironmentOverride` 证明 |
| 启用 OIDC 时 discovery 为启动期网络调用（30s 超时，失败退出）；namespace decoder 切换为 Session 版 | `cmd/server/main.go` diff |
| **上游 v1 本无强制认证**（PostAuthMiddlewares 仅 feature-flag，`app/common/server.go:33-41`）→ 启用后 v1/v3 除豁免前缀全部 401 | 行为面变化，见 Task 4 |

### 前端（front/ 子模块 flexprice-front）

| 行为 | 证据 |
|---|---|
| SSO 按钮仅当 `VITE_OIDC_LOGIN_URL` 非空时渲染 | `front/src/pages/auth/LoginForm.tsx:168-177` |
| 点击生成 16 字节 nonce 写 sessionStorage（`oidc_login_pending`/`oidc_login_state`），**整页跳转** `${oidcLoginUrl}?state=<nonce>` | `front/src/pages/auth/OidcSignin.tsx:51-58` |
| 回调页 `/auth/callback` 只读 URL fragment；要求本 tab pending 标记 + nonce 与 fragment `state` 一致，否则 unsolicitedToken；成功写 `localStorage.token = {token, tenant_id}`、清 fragment、跳首页 | `front/src/pages/auth/SamlCallback.tsx:38-132` |
| 后续请求 axios 拦截器带 `Authorization: Bearer`；401 → `AuthService.logout()`（清 localStorage、跳 /login） | `front/src/core/axios/config.ts:36-82` |
| 用户信息 `/users/me` 在本地模式是纯垫片（`UserApi.me()` 返回本地单管理员，不打后端）→ SSO E2E 可闭环 | `front/src/api/UserApi.ts:88-91` |
| 本地 `.env` 未配 `VITE_OIDC_LOGIN_URL`，密码登录是合成 token 垫片（`local-openmeter-session`）——认证启用后该垫片 token 会被后端 401 | `front/src/core/services/platform/localPlatform.ts:15` |
| e2e Playwright 走真后端、真实 UI 登录建 storageState；**无任何 SSO 用例** | `front/e2e/support/auth.setup.ts`，`front/playwright.config.ts` |

### 已有测试盘点（不重复建设）

- `openmeter/auth/handler_test.go`：fake IdP（httptest + RS256）覆盖快乐路径、state 不匹配/缺失、伪造 id_token、token roundtrip/错 secret/缺 userId。复用其 `newFakeIDP`/`newTestHandler`/`startLogin`/`randomSecret`。
- `openmeter/auth/middleware_test.go`：`TestSessionMiddleware` 6 子测试（缺 bearer/Basic/垃圾 token/无 org/合法 token/portal 豁免）。
- `app/config/auth_test.go`、`openmeter/namespace/namespacedriver/session_test.go`、`openmeter/server/server_test.go` 的 `TestAuthRoutes`/`TestSessionAuthRoutes`（getTestServer 为 noop 全栈，无需 DB）。

---

## 二、风险登记表（测试须回答的问题)

| # | 风险 | 归属任务 |
|---|---|---|
| R1 | 会话 JWT 算法混淆（none/HS512）、过期、错签发者、篡改是否全被拒；⚠️ 未知 claim 字段实测**会被接受**（jwt/v5 `WithStrictDecoding` 不做 claim-set pinning） | Task 1 |
| R2 | IdP 故障/无 id_token 时回调是否 502 而非误发 token；state 是否真单次使用（浏览器 cookie 语义） | Task 2 |
| R3 | organization/owner 优先级与两者皆空的行为（签发但中间件 401 → "登录成功却是死端"） | Task 2 |
| R4 | 豁免前缀边界（`/api/v1/portal` 无尾斜杠、`/api/v1/portalx`）是否如设计排除 | Task 3、4 |
| R5 | **行为面**：启用认证后匿名 v1（ingest 等）全 401；禁用时全通（防误伤回归锚点） | Task 4 |
| R6 | 前端回调页防御（nonce 绑定、pending 标记、fragment 清理）在真实组件层是否成立 | Task 5 |
| R7 | 浏览器全链路：按钮 → IdP → 回调 → 落地仪表盘 → 带 token 的 API 调用成功；回调 URL 重放到新 tab 被拒 | Task 6 |
| R8 | 真实 Casdoor 的 discovery/claims/redirect 注册与本地 fake 的差异 | Task 7（手工） |
| R9 | 遗留设计决策（无 PKCE、无 aud/jti、30 天不吊销、x-api-key 劫持、空 org 死端、垫片 token 与强制认证的冲突、**OIDC 登录者一律 org admin 无角色映射**——测试已钉住该行为，是否保留属 Task 8 决策） | Task 8（决策清单） |

---

## 三、分层总览

| 层 | 对象 | 基础设施 | 现状 |
|---|---|---|---|
| L1 后端单测 | session_token.go / handler.go / middleware.go | fake IdP (httptest) | 有骨架，本方案补边界 |
| L2 后端集成 | server.go 挂载 + 行为面 | getTestServer noop 全栈 | 有 401/200 锚点，补豁免边界与 ingest 锁死 |
| L3 前端单测 | OidcSignin / SamlCallback | vitest + jsdom（已配好） | **空白**，本方案新建 |
| L4 浏览器 E2E | 前后端契约全链路 | Node fake IdP + Playwright + 真 OpenMeter 实例 | **空白**，本方案新建 |
| L5 手工联调 | 真实 Casdoor | Casdoor 实例（待办清单在记忆） | 本方案给 checklist |

---

### Task 1: 后端单测 — 会话 JWT 安全边界

**Files:**
- Create: `openmeter/auth/session_token_test.go`（同包新文件，不与 handler_test.go 冲突）

**Interfaces:**
- Consumes: `NewSessionTokenIssuer(secret string, expire time.Duration) (*SessionTokenIssuer, error)`、`(*SessionTokenIssuer).Issue(IssueSessionTokenInput) (string, error)`、`(*SessionTokenIssuer).Verify(string) (*SessionTokenClaims, error)`（`openmeter/auth/session_token.go`）
- Produces: 无（纯测试）

- [ ] **Step 1: 写测试文件**

```go
package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// mintSessionToken hand-builds a session-shaped JWT so tests can vary a single
// property (algorithm, issuer, expiry) while keeping everything else valid.
func mintSessionToken(t *testing.T, secret string, signingMethod jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()

	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString(key)
	require.NoError(t, err)

	return tokenString
}

func TestSessionTokenIssuerVerifyRejects(t *testing.T) {
	secret := randomSecret(t)

	issuer, err := NewSessionTokenIssuer(secret, time.Hour)
	require.NoError(t, err)

	now := time.Now()

	t.Run("expired token", func(t *testing.T) {
		expired, err := NewSessionTokenIssuer(secret, -time.Minute)
		require.NoError(t, err)

		token, err := expired.Issue(IssueSessionTokenInput{UserID: "user-123", Organization: "built-in"})
		require.NoError(t, err)

		_, err = issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss":          "not-openmeter",
			"sub":          "user-123",
			"iat":          now.Unix(),
			"exp":          now.Add(time.Hour).Unix(),
			"organization": "built-in",
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("alg none", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodNone, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}, jwt.UnsafeAllowNoneSignatureType)

		_, err = issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("HS512 with the same secret", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS512, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("missing expiry", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("tampered payload", func(t *testing.T) {
		token, err := issuer.Issue(IssueSessionTokenInput{UserID: "user-123", Organization: "built-in"})
		require.NoError(t, err)

		head, payload, _, ok := strings.Cut(token, ".")
		require.True(t, ok)

		// Flip the first payload character so the signature no longer matches.
		flipped := []byte(payload)
		if flipped[0] == 'e' {
			flipped[0] = 'f'
		} else {
			flipped[0] = 'e'
		}

		_, err = issuer.Verify(head + "." + string(flipped) + "." + strings.Split(token, ".")[2])
		require.Error(t, err)
	})

	t.Run("unknown fields (strict decoding pins the claim set)", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss":     SessionTokenIssuerName,
			"sub":     "user-123",
			"iat":     now.Unix(),
			"exp":     now.Add(time.Hour).Unix(),
			"surprise": "future-claim",
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("empty organization verifies (enforcement is the middleware's job)", func(t *testing.T) {
		token, err := issuer.Issue(IssueSessionTokenInput{UserID: "user-123"})
		require.NoError(t, err)

		claims, err := issuer.Verify(token)
		require.NoError(t, err)
		require.Empty(t, claims.Organization)
	})
}
```

- [ ] **Step 2: 运行并确认通过**

Run: `cd /Users/wuyongjun/trea/openmeter && go test ./openmeter/auth/ -run TestSessionTokenIssuerVerifyRejects -v`
Expected: 全部子测试 PASS（当前实现的 `WithValidMethods`/`WithExpirationRequired`/`WithIssuer`/`WithStrictDecoding` 应满足以上全部断言；任何 FAIL 即发现真实缺陷，记录到风险表再修）。

- [ ] **Step 3: Commit**

```bash
git add openmeter/auth/session_token_test.go
git commit -m "test(auth): pin session JWT verification boundaries"
```

---

### Task 2: 后端单测 — Callback 失败路径、state 单次使用、组织 claim 语义

**Files:**
- Modify: `openmeter/auth/handler_test.go`（fakeIDP 加测试开关 + 新增 4 个测试函数）

**Interfaces:**
- Consumes: 现有 `fakeIDP`/`newTestHandler`/`startLogin`/`randomSecret`（handler_test.go 内）
- Produces: `fakeIDP` 新增字段 `failTokenExchange bool`、`omitIDToken bool`、`organization string`、`noOrganization bool`（后续任务不依赖，但同名约定供评审）

- [ ] **Step 1: 给 fakeIDP 加开关**

`fakeIDP` struct 追加字段（`openmeter/auth/handler_test.go:44-51`）：

```go
type fakeIDP struct {
	*httptest.Server

	jwksKey    *rsa.PrivateKey
	signingKey *rsa.PrivateKey

	// failTokenExchange makes the token endpoint answer 500, simulating a
	// provider outage during the code exchange.
	failTokenExchange bool

	// omitIDToken makes the token response omit the id_token field.
	omitIDToken bool

	// organization adds an "organization" claim alongside "owner" when set;
	// noOrganization strips the "owner" claim instead.
	organization    string
	noOrganization  bool
}
```

`serveHTTP` 的 token endpoint 分支（`handler_test.go:90`）改造为：

```go
	case "/api/login/oauth/access_token":
		if idp.failTokenExchange {
			http.Error(w, "upstream unavailable", http.StatusInternalServerError)

			return
		}

		now := time.Now()

		claims := jwt.MapClaims{
			"iss":   idp.URL,
			"aud":   testClientID,
			"sub":   "user-123",
			"iat":   now.Unix(),
			"exp":   now.Add(time.Hour).Unix(),
			"email": "user@example.com",
			"name":  "Test User",
			"owner": "built-in",
		}
		if idp.organization != "" {
			claims["organization"] = idp.organization
		}
		if idp.noOrganization {
			delete(claims, "owner")
		}

		idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		idToken.Header["kid"] = "test-key"

		if idp.omitIDToken {
			writeJSON(w, map[string]any{
				"access_token": randomHex(),
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

			return
		}

		signed, err := idToken.SignedString(idp.signingKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		writeJSON(w, map[string]any{
			"access_token": randomHex(),
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     signed,
		})
```

注意：`newFakeIDP` 返回的是 `&fakeIDP{...}`，开关在测试里直接对返回值赋值即可（`idp.failTokenExchange = true`），无需改构造器。

- [ ] **Step 2: 追加测试函数（文件末尾）**

```go
func TestHandlerCallbackTokenExchangeFailure(t *testing.T) {
	// given a provider whose token endpoint is down
	idp := newFakeIDP(t)
	idp.failTokenExchange = true

	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, cookies := startLogin(t, client, srv.URL, "")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then the callback fails with a bad gateway, not a session token
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestHandlerCallbackTokenResponseWithoutIDToken(t *testing.T) {
	// given a provider whose token response carries no id_token
	idp := newFakeIDP(t)
	idp.omitIDToken = true

	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, cookies := startLogin(t, client, srv.URL, "")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestHandlerCallbackStateIsSingleUse(t *testing.T) {
	// given a browser that keeps cookies in a jar (honouring MaxAge=-1 deletes)
	idp := newFakeIDP(t)
	handler, tokens := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, _ := startLogin(t, client, srv.URL, "")

	callbackURL := srv.URL + "/auth/oidc/callback?code=test-code&state=" + url.QueryEscape(location.Query().Get("state"))

	// when the callback is completed, then replayed with the same URL
	first, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer first.Body.Close()
	require.Equal(t, http.StatusFound, first.StatusCode)

	second, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer second.Body.Close()

	// then the replay is rejected: the browser dropped the single-use state cookie
	require.Equal(t, http.StatusBadRequest, second.StatusCode)
}

func TestHandlerCallbackOrganizationClaims(t *testing.T) {
	t.Run("organization claim wins over owner", func(t *testing.T) {
		idp := newFakeIDP(t)
		idp.organization = "acme-org"

		handler, tokens := newTestHandler(t, idp)

		router := chi.NewRouter()
		require.NoError(t, handler.RegisterRoutes(router))

		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)

		client := &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}

		location, cookies := startLogin(t, client, srv.URL, "")

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
		require.NoError(t, err)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		redirect, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)

		fragment, err := url.ParseQuery(redirect.Fragment)
		require.NoError(t, err)
		require.Equal(t, "acme-org", fragment.Get("tenant_id"))

		claims, err := tokens.Verify(fragment.Get("token"))
		require.NoError(t, err)
		require.Equal(t, "acme-org", claims.Organization)
	})

	t.Run("no organization issues a token without tenant_id", func(t *testing.T) {
		// Documents the current dead-end: the login "succeeds" but the
		// session middleware will 401 every API call. Tracked in the risk
		// register (R3) — if fixed to reject at the callback, update this.
		idp := newFakeIDP(t)
		idp.noOrganization = true

		handler, tokens := newTestHandler(t, idp)

		router := chi.NewRouter()
		require.NoError(t, handler.RegisterRoutes(router))

		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)

		client := &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}

		location, cookies := startLogin(t, client, srv.URL, "")

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
		require.NoError(t, err)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusFound, resp.StatusCode)

		redirect, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)

		fragment, err := url.ParseQuery(redirect.Fragment)
		require.NoError(t, err)
		require.Empty(t, fragment.Get("tenant_id"))

		claims, err := tokens.Verify(fragment.Get("token"))
		require.NoError(t, err)
		require.Empty(t, claims.Organization)
	})
}
```

import 块需补 `"net/http/cookiejar"`。

- [ ] **Step 3: 运行**

Run: `cd /Users/wuyongjun/trea/openmeter && go test ./openmeter/auth/ -run 'TestHandlerCallback(TokenExchangeFailure|TokenResponseWithoutIDToken|StateIsSingleUse|OrganizationClaims)' -v`
Expected: PASS。`TestHandlerCallbackStateIsSingleUse` 若 FAIL 说明 cookie 清除语义有缺陷（真问题，停下记录）。

- [ ] **Step 4: 回归全包**

Run: `go test ./openmeter/auth/ -v`
Expected: 既有测试全部仍 PASS。

- [ ] **Step 5: Commit**

```bash
git add openmeter/auth/handler_test.go
git commit -m "test(auth): cover callback failure paths, state single-use, org claim precedence"
```

---

### Task 3: 后端单测 — 豁免前缀边界

**Files:**
- Modify: `openmeter/auth/middleware_test.go`（追加一个自包含测试函数，不依赖文件内既有 helper）

**Interfaces:**
- Consumes: `NewSessionMiddleware(SessionMiddlewareConfig) (func(http.Handler) http.Handler, error)`（middleware.go）
- Produces: 无

- [ ] **Step 1: 追加测试**

```go
func TestSessionMiddlewareExemptPrefixBoundaries(t *testing.T) {
	// given the production exemption list shape (trailing-slash portal prefix)
	tokens, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
	require.NoError(t, err)

	middleware, err := NewSessionMiddleware(SessionMiddlewareConfig{
		Tokens:             tokens,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ExemptPathPrefixes: []string{"/api/v1/portal/", "/api/swagger.json"},
	})
	require.NoError(t, err)

	reached := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true

		w.WriteHeader(http.StatusOK)
	}))

	t.Run("portal without trailing slash is not exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portal", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.False(t, reached)
	})

	t.Run("portal sibling path is not exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portalx/tokens", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.False(t, reached)
	})

	t.Run("portal subtree is exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portal/apikeys", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, reached)
	})

	t.Run("swagger spec is exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/swagger.json", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, reached)
	})
}
```

import 块需补 `"io"`（若 middleware_test.go 尚未引入）。

- [ ] **Step 2: 运行**

Run: `cd /Users/wuyongjun/trea/openmeter && go test ./openmeter/auth/ -run TestSessionMiddlewareExemptPrefixBoundaries -v`
Expected: 4 子测试 PASS（HasPrefix 语义下 `/api/v1/portal`、`/api/v1/portalx` 均不匹配 `/api/v1/portal/`）。

- [ ] **Step 3: Commit**

```bash
git add openmeter/auth/middleware_test.go
git commit -m "test(auth): pin exempt path prefix boundaries"
```

---

### Task 4: 后端集成锚点 — 认证开关的行为面回归

**Files:**
- Modify: `openmeter/server/server_test.go`（在 `TestSessionAuthRoutes` 后追加 `TestSessionAuthBlastRadius`）

**Interfaces:**
- Consumes: `getTestServer(t, opts ...func(*Config))`（server_test.go:706）、`auth.SessionMiddlewareConfig{Tokens, Logger}`、`DefaultNamespace`（文件内已有引用）
- Produces: 无

- [ ] **Step 1: 追加测试**

```go
// TestSessionAuthBlastRadius pins the intended behavior change of enabling
// session enforcement: the previously anonymous v1 surface (including event
// ingestion) starts requiring a session token, and disabling enforcement
// restores anonymous access. If a test here fails after touching exempt
// prefixes or middleware ordering, the blast radius moved on purpose — update
// this test and the deployment notes together.
func TestSessionAuthBlastRadius(t *testing.T) {
	randomTestSecret := func() string {
		buf := make([]byte, 32)
		_, err := rand.Read(buf)
		require.NoError(t, err)

		return hex.EncodeToString(buf)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tokens, err := auth.NewSessionTokenIssuer(randomTestSecret(), time.Hour)
	require.NoError(t, err)

	t.Run("ingestion requires a session when enforcement is on", func(t *testing.T) {
		testServer, _ := getTestServer(t, func(c *Config) {
			c.SessionAuth = auth.SessionMiddlewareConfig{Tokens: tokens, Logger: logger}
		})

		w := httptest.NewRecorder()
		testServer.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{"specversion":"1.0"}`)))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("portal prefix without trailing slash still requires a session", func(t *testing.T) {
		testServer, _ := getTestServer(t, func(c *Config) {
			c.SessionAuth = auth.SessionMiddlewareConfig{Tokens: tokens, Logger: logger}
		})

		w := httptest.NewRecorder()
		testServer.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/portal", nil))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("disabling enforcement restores the anonymous surface", func(t *testing.T) {
		testServer, _ := getTestServer(t)

		w := httptest.NewRecorder()
		testServer.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil))

		require.Equal(t, http.StatusOK, w.Code)
	})
}
```

import 块需补 `"strings"`（若未有）。

- [ ] **Step 2: 运行**

Run: `cd /Users/wuyongjun/trea/openmeter && go test -tags=dynamic ./openmeter/server/ -run 'TestSessionAuthBlastRadius|TestSessionAuthRoutes|TestAuthRoutes' -v`
Expected: 全部 PASS。

- [ ] **Step 3: 全包回归**

Run: `go test -tags=dynamic ./openmeter/server/...`
Expected: PASS（认证关闭路径不受影响）。

- [ ] **Step 4: Commit**

```bash
git add openmeter/server/server_test.go
git commit -m "test(server): pin session enforcement blast radius on the v1 surface"
```

---

### Task 5: 前端组件单测 — OidcSignin 与 SamlCallback 的 OIDC 分支

**Files:**
- Create: `front/src/pages/auth/OidcSignin.test.tsx`
- Create: `front/src/pages/auth/SamlCallback.test.tsx`

**Interfaces:**
- Consumes: `OidcSignin`（默认导出，`OidcSignin.tsx`）、`SamlCallback`（默认导出，`SamlCallback.tsx`）、常量 `OIDC_PENDING_KEY='oidc_login_pending'`、`OIDC_STATE_KEY='oidc_login_state'`；vitest globals + jsdom（`vitest.config.ts` 已配 url=localhost:3000）
- Produces: 无

- [ ] **Step 1: 写 OidcSignin.test.tsx**

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const navigateMock = vi.fn();

vi.mock('react-router', () => ({ useNavigate: () => navigateMock }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

// jsdom cannot navigate; replace location with a writable stub to capture the
// full-page redirect the component performs.
const originalLocation = window.location;

describe('OidcSignin', () => {
	beforeEach(() => {
		vi.resetModules();
		vi.stubEnv('VITE_OIDC_LOGIN_URL', 'http://backend.test/auth/oidc/login');
		sessionStorage.clear();

		// @ts-expect-error -- jsdom navigation is not implemented
		delete window.location;
		window.location = { href: '' } as unknown as Location;
	});

	afterEach(() => {
		window.location = originalLocation;
		vi.unstubAllEnvs();
	});

	it('stores a fresh nonce and redirects to the backend login', async () => {
		const { default: OidcSignin } = await import('./OidcSignin');

		render(<OidcSignin />);
		await userEvent.click(screen.getByRole('button'));

		const nonce = sessionStorage.getItem('oidc_login_state');
		expect(sessionStorage.getItem('oidc_login_pending')).toBe('true');
		expect(nonce).toMatch(/^[0-9a-f]{32}$/);
		expect(window.location.href).toBe('http://backend.test/auth/oidc/login?state=' + nonce);
	});

	it('generates a different nonce per click', async () => {
		const { default: OidcSignin } = await import('./OidcSignin');

		render(<OidcSignin />);
		await userEvent.click(screen.getByRole('button'));
		const first = sessionStorage.getItem('oidc_login_state');
		await userEvent.click(screen.getByRole('button'));
		const second = sessionStorage.getItem('oidc_login_state');

		expect(first).not.toEqual(second);
	});
});
```

- [ ] **Step 2: 写 SamlCallback.test.tsx**

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const navigateMock = vi.fn();

vi.mock('react-router', () => ({ useNavigate: () => navigateMock }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

import SamlCallback from './SamlCallback';
import { OIDC_PENDING_KEY, OIDC_STATE_KEY } from './OidcSignin';

const armOidcLogin = (nonce: string) => {
	sessionStorage.setItem(OIDC_PENDING_KEY, 'true');
	sessionStorage.setItem(OIDC_STATE_KEY, nonce);
};

describe('SamlCallback OIDC branch', () => {
	beforeEach(() => {
		sessionStorage.clear();
		localStorage.clear();
		window.location.hash = '';
		navigateMock.mockClear();
	});

	afterEach(() => {
		window.location.hash = '';
	});

	it('stores the token and navigates home on a valid nonce', async () => {
		armOidcLogin('nonce-123');
		window.location.hash = '#token=jwt-value&tenant_id=acme&state=nonce-123';

		render(<SamlCallback />);

		await waitFor(() => {
			expect(localStorage.getItem('token')).toBe(JSON.stringify({ token: 'jwt-value', tenant_id: 'acme' }));
		});
		await waitFor(() => {
			expect(navigateMock).toHaveBeenCalledWith('/', { replace: true });
		});
		expect(window.location.hash).toBe('');
	});

	it('rejects a fragment whose nonce does not match', async () => {
		armOidcLogin('nonce-123');
		window.location.hash = '#token=jwt-value&tenant_id=acme&state=other-nonce';

		render(<SamlCallback />);

		await waitFor(() => {
			expect(screen.getByText('sso.unsolicitedToken')).toBeInTheDocument();
		});
		expect(localStorage.getItem('token')).toBeNull();
		expect(navigateMock).not.toHaveBeenCalled();
		// markers are consumed either way: no replay into this tab
		expect(sessionStorage.getItem(OIDC_PENDING_KEY)).toBeNull();
		expect(sessionStorage.getItem(OIDC_STATE_KEY)).toBeNull();
	});

	it('rejects a callback this tab never started', async () => {
		window.location.hash = '#token=jwt-value&tenant_id=acme&state=nonce-123';

		render(<SamlCallback />);

		await waitFor(() => {
			expect(screen.getByText('sso.unsolicitedToken')).toBeInTheDocument();
		});
		expect(localStorage.getItem('token')).toBeNull();
	});

	it('reports a fragment without a token', async () => {
		armOidcLogin('nonce-123');
		window.location.hash = '#tenant_id=acme&state=nonce-123';

		render(<SamlCallback />);

		await waitFor(() => {
			expect(screen.getByText('sso.missingToken')).toBeInTheDocument();
		});
		expect(localStorage.getItem('token')).toBeNull();
	});

	it('pins current behavior: empty tenant_id is stored as-is (risk R6 follow-up)', async () => {
		armOidcLogin('nonce-123');
		window.location.hash = '#token=jwt-value&state=nonce-123';

		render(<SamlCallback />);

		await waitFor(() => {
			expect(localStorage.getItem('token')).toBe(JSON.stringify({ token: 'jwt-value', tenant_id: '' }));
		});
	});
});
```

- [ ] **Step 3: 运行**

Run: `cd /Users/wuyongjun/trea/openmeter/front && npx vitest run src/pages/auth`
Expected: 全部 PASS。若 `@/components/atoms`（PageLoader）的 barrel import 拖入过重依赖导致报错，允许在测试里 `vi.mock('@/components/atoms', () => ({ PageLoader: () => null }))`。

- [ ] **Step 4: Commit**

```bash
git add front/src/pages/auth/OidcSignin.test.tsx front/src/pages/auth/SamlCallback.test.tsx
git commit -m "test(front): cover OIDC sign-in redirect and callback page guards"
```

---

### Task 6: 浏览器 E2E — fake IdP + SSO 全链路

**前置条件（人工，一次性）：**
- 本地 PostgreSQL 已按启动手册就绪（runbook：Homebrew go + 本地依赖栈）。
- 常驻的 air/8888 实例先停掉，E2E 期间用下面的命令临时占住 8888（front 的 vite 代理写死指向 8888）。

**Files:**
- Create: `front/e2e/support/fake-oidp.mjs`
- Create: `front/e2e/sso/login.spec.ts`
- Modify: `front/playwright.config.ts`（E2E_SSO=1 门控的 sso project + fake IdP webServer）
- Modify: `front/package.json`（script `test:e2e:sso`）

**Interfaces:**
- Consumes: 后端环境变量 `AUTH_OIDC_*`/`AUTH_TOKENSECRET`（`app/config/auth.go` 已被 `TestAuthConfigurationEnvironmentOverride` 证明）；Playwright `webServer` 数组与条件 project 模式（config 内 rbac/visual 同款）
- Produces: fake IdP 固定签发 `organization=acme` 的 id_token → 后端会话 org=`acme` → 前端 `tenant_id=acme`

- [ ] **Step 1: 写 fake-oidp.mjs（Node 零依赖）**

```js
// Minimal OIDC provider for the SSO e2e profile. Serves discovery, JWKS,
// authorize (auto-approve) and token endpoints; signs RS256 id_tokens with an
// in-process key. Claims are fixed via env so the backend under test lands in
// a known organization.
import { createServer } from 'node:http';
import { createSign, generateKeyPairSync, randomUUID } from 'node:crypto';

const issuer = process.env.FAKE_OIDP_URL ?? 'http://127.0.0.1:9401';
const clientId = process.env.FAKE_OIDP_CLIENT_ID ?? 'openmeter';
const organization = process.env.FAKE_OIDP_ORGANIZATION ?? 'acme';

const { publicKey, privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
const jwk = publicKey.export({ format: 'jwk' });
const privatePem = privateKey.export({ format: 'pem', type: 'pkcs8' });

const b64u = (value) => Buffer.from(value).toString('base64url');

function signIDToken() {
	const now = Math.floor(Date.now() / 1000);
	const header = b64u(JSON.stringify({ alg: 'RS256', typ: 'JWT', kid: 'e2e-key' }));
	const payload = b64u(
		JSON.stringify({
			iss: issuer,
			aud: clientId,
			sub: `e2e-user-${randomUUID()}`,
			iat: now,
			exp: now + 3600,
			email: 'e2e@example.com',
			name: 'E2E User',
			organization,
		}),
	);
	const input = `${header}.${payload}`;
	const signature = createSign('RSA-SHA256').update(input).sign(privatePem, 'base64url');

	return `${input}.${signature}`;
}

function json(res, body) {
	res.writeHead(200, { 'Content-Type': 'application/json' });
	res.end(JSON.stringify(body));
}

createServer((req, res) => {
	const parsed = new URL(req.url, issuer);

	switch (parsed.pathname) {
		case '/.well-known/openid-configuration':
			return json(res, {
				issuer,
				authorization_endpoint: `${issuer}/login/oauth/authorize`,
				token_endpoint: `${issuer}/api/login/oauth/access_token`,
				jwks_uri: `${issuer}/api/certs`,
				id_token_signing_alg_values_supported: ['RS256'],
			});
		case '/api/certs':
			return json(res, {
				keys: [{ kty: jwk.kty, alg: 'RS256', use: 'sig', kid: 'e2e-key', n: jwk.n, e: jwk.e }],
			});
		case '/login/oauth/authorize': {
			const redirect = new URL(parsed.searchParams.get('redirect_uri'));
			redirect.searchParams.set('code', 'e2e-code');
			redirect.searchParams.set('state', parsed.searchParams.get('state') ?? '');

			res.writeHead(302, { Location: redirect.toString() });

			return res.end();
		}
		case '/api/login/oauth/access_token':
			return json(res, {
				access_token: randomUUID(),
				token_type: 'Bearer',
				expires_in: 3600,
				id_token: signIDToken(),
			});
		default:
			res.writeHead(404);
			return res.end();
	}
}).listen(9401, '127.0.0.1', () => {
	console.log(`fake OIDC provider on ${issuer}`);
});
```

- [ ] **Step 2: 写 sso/login.spec.ts**

```ts
import { expect, test } from '@playwright/test';

// The SSO profile has no shared storage state: every test walks the real flow.
test.use({ storageState: { cookies: [], origins: [] } });

test('SSO login lands on the dashboard with an authenticated API call', async ({ page }) => {
	await page.goto('/login');

	await page.getByRole('button', { name: /sso/i }).click();

	// Fake IdP auto-approves, so the browser chains:
	// authorize → backend callback → /auth/callback#fragment → home.
	await page.waitForURL((url) => !url.pathname.startsWith('/login') && !url.pathname.startsWith('/auth'), {
		timeout: 15_000,
	});

	// The callback page must scrub the fragment out of the URL.
	expect(page.url()).not.toContain('#token');

	// The stored session is the contract everything else reads.
	const stored = await page.evaluate(() => localStorage.getItem('token'));
	expect(stored).not.toBeNull();
	const parsed = JSON.parse(stored!) as { token: string; tenant_id: string };
	expect(parsed.token).not.toBe('local-openmeter-session'); // the shim token, not a real session
	expect(parsed.tenant_id).toBe('acme');

	// The dashboard actually gets data through the session token: wait for a
	// successful same-origin API response after login.
	const apiResponse = await page.waitForResponse(
		(resp) => resp.url().includes('/openmeter/') && resp.request().method() === 'GET' && resp.ok(),
		{ timeout: 15_000 },
	);
	expect(apiResponse.status()).toBe(200);
});

test('replaying the callback URL in a fresh tab is rejected', async ({ browser }) => {
	const context = await browser.newContext();
	const page = await context.newPage();

	// No sessionStorage in this tab: no pending marker, so the callback page
	// must refuse to adopt whatever the URL carries.
	await page.goto('/auth/callback#token=attacker-token&tenant_id=acme&state=forged');

	await page.waitForURL('**/auth/callback**');

	const stored = await page.evaluate(() => localStorage.getItem('token'));
	expect(stored).toBeNull();

	await context.close();
});
```

- [ ] **Step 3: playwright.config.ts 增加 sso project 与 fake IdP webServer**

按 config 内既有条件 project（rbac/visual）的写法追加。要点（与现有字段名对齐后再提交）：

```ts
const ssoEnabled = !!process.env.E2E_SSO;

// projects 数组追加：
...(ssoEnabled
	? [
			{
				name: 'sso',
				testDir: 'e2e/sso',
				use: { ...devices['Desktop Chrome'], baseURL },
				dependencies: [] as string[],
			},
		]
	: []),

// webServer 数组追加（fake IdP 独立端口，后端由执行者按 runbook 手动拉起）：
...(ssoEnabled
	? [
			{
				command: 'node e2e/support/fake-oidp.mjs',
				url: 'http://127.0.0.1:9401/.well-known/openid-configuration',
				reuseExistingServer: !process.env.CI,
				env: { FAKE_OIDP_URL: 'http://127.0.0.1:9401' },
			},
		]
	: []),
```

- [ ] **Step 4: package.json 增加 script**

```json
"test:e2e:sso": "E2E_SSO=1 playwright test --project=sso"
```

- [ ] **Step 5: 启动带认证的后端（执行者按此 runbook 操作）**

```bash
cd /Users/wuyongjun/trea/openmeter
# 先停掉常驻 air 实例（占用 :8888 的那个），并确认本地 PostgreSQL 在跑
AUTH_OIDC_ENABLED=true \
AUTH_OIDC_ISSUER=http://127.0.0.1:9401 \
AUTH_OIDC_CLIENTID=openmeter \
AUTH_OIDC_CLIENTSECRET=e2e-client-secret \
AUTH_OIDC_REDIRECTURL=http://127.0.0.1:8888/auth/oidc/callback \
AUTH_OIDC_DASHBOARDURL=http://localhost:3000/auth/callback \
AUTH_TOKENSECRET=$(openssl rand -hex 32) \
go run ./cmd/server --config config.yaml
```

预期：启动日志无 `failed to create OIDC auth handler`（discovery 成功）。`curl -s http://127.0.0.1:9401/.well-known/openid-configuration | head -c 200` 可达。

- [ ] **Step 6: 启动带 SSO 按钮的前端并跑 E2E**

```bash
cd /Users/wuyongjun/trea/openmeter/front
VITE_OIDC_LOGIN_URL=http://127.0.0.1:8888/auth/oidc/login npm run dev   # 终端 A
E2E_BASE_URL=http://localhost:3000 npx playwright test --project=sso    # 终端 B
```

Expected: 2 个用例 PASS。失败时按 Playwright trace 截图定位（不要用 ZCode 内置浏览器复现下结论）。

- [ ] **Step 7: 正向冒烟（人眼确认）**

浏览器开 `http://localhost:3000/login` → 点 "Continue with SSO" → 自动回跳 → 仪表盘出现、网络面板可见带 `Authorization: Bearer` 的 200 请求。

- [ ] **Step 8: Commit**

```bash
git add front/e2e/support/fake-oidp.mjs front/e2e/sso/login.spec.ts front/playwright.config.ts front/package.json
git commit -m "test(front): browser e2e for the OIDC SSO flow against a fake provider"
```

---

### Task 7: 手工验收清单 — 真实 Casdoor 联调

fake IdP 与真实 Casdoor 的差异点集中在：discovery 元数据完整性、claims 形状（`owner` vs `organization`）、redirect 注册严格性、authorize 页面交互。逐项勾选（需要记忆中待办的 issuer/clientId/clientSecret）：

| # | 检查项 | 通过标准 |
|---|---|---|
| 1 | 后端启动：`AUTH_OIDC_ISSUER=https://<casdoor>` discovery 成功 | 启动无 `failed to create OIDC auth handler`；30s 内起来 |
| 2 | 登录跳转 | `/auth/oidc/login` 302 到 Casdoor 授权页，state cookie 已种 |
| 3 | Casdoor 授权后回跳 | `302 → dashboardURL#token=…&tenant_id=…`，无 400/502 |
| 4 | claims 形状 | 若 Casdoor 只给 `owner`（旧版）或只给 `organization`（新版），tenant_id 均正确（后端两者都收） |
| 5 | 前端落地 | `/auth/callback` 验 nonce 通过 → 进仪表盘 → 数据加载（Bearer 200） |
| 6 | 错误 clientSecret | 后端 token exchange 失败 → 回调 502，浏览器不死循环 |
| 7 | 401 链路 | 手工清掉 localStorage 后刷新 → API 401 → 前端自动回 /login |
| 8 | 大小写 org | Casdoor org 含大写（如 `Acme`）→ 后端 namespace 落地为 `acme`（v1/v3 数据可见性正确） |
| 9 | 多 tab 约束 | 登录 URL 复制到第二个 tab 打开回调 → unsolicitedToken，不串会话 |
| 10 | state 时效 | login 后等 10 分钟再走完回调 → 400（state cookie 过期） |
| 11 | HTTPS 部署预检 | 反代后 cookie `Secure`、fragment 不出现在访问日志（抽查 nginx log） |

- [ ] 完成后把结果（含 Casdoor 版本与 claims 截图）回填到任务工单。

---

### Task 8: 遗留风险决策清单（不阻塞测试落地）

以下设计现状已被测试/审查确认，是否接受或修复需产品/安全决策。逐项标记：接受 / 排期修复：

| # | 现状 | 影响 | 建议 |
|---|---|---|---|
| 1 | OIDC 无 PKCE（仅 client_secret） | secret 泄露或 redirect 校验不严时授权码可被拦截 | 部署在可信网络或补 PKCE |
| 2 | 会话 JWT 无 aud/jti、无吊销、默认 720h | token 泄露最长 30 天可用 | 缩短 tokenExpiration 或加黑名单 |
| 3 | 所有 OIDC 登录者硬编码 org admin | 无角色映射，任何 org 成员都是管理员 | 角色映射任务（已在 backlog） |
| 4 | 空 org 用户登录"成功"但 API 全 401 | 死端体验 | Task 2 已钉住行为；建议 callback 直接拒绝 |
| 5 | 前端 `x-api-key` 若存在会覆盖会话 Authorization 头 | localStorage 被污染时认证静默失效 | 前端修复任务 |
| 6 | 前端密码登录垫片 token（`local-openmeter-session`）与强制认证并存 | 后端启用认证后垫片登录的用户全 401 → 登出循环 | E2E profile 已隔离；前端需在认证启用时隐藏密码登录 |
| 7 | JWT Verify 无 clock leeway | 多副本时钟偏移时边缘 401 | 如出现加 `WithLeeway(30s)` |
| 8 | 回调页清 URL 连 query 一起丢 | 未来若有 query 参数需求会踩坑 | 记录即可 |

- [ ] 决策结果回填本表并同步到工单。

---

## Self-Review 记录

- **覆盖度**：风险表 R1→Task1、R2/R3→Task2、R4→Task3/4、R5→Task4、R6→Task5、R7→Task6、R8→Task7、R9→Task8，无遗漏。
- **占位符扫描**：所有测试代码为完整可粘贴内容；Task 6 Step 3 的 playwright.config.ts 改动以"要点+与现有字段名对齐"给出（config 内部结构以现场为准），执行者需读文件后套用该模式，这是唯一非逐字改动。
- **符号一致性**：`fakeIDP` 字段名、`OIDC_PENDING_KEY`/`OIDC_STATE_KEY` 常量名、`getTestServer`/`DefaultNamespace`、后端 env 前缀 `AUTH_OIDC_*`/`AUTH_TOKENSECRET` 均与源码核对过。
