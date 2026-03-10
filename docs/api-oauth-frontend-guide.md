# Google OAuth 前端接入指南

## Google Client ID

```
358687166908-pp8q17uu638a4boh8lptvhv34q94qp2a.apps.googleusercontent.com
```

## 接入流程

```
用户点击「Google 登录」
        ↓
前端调用 Google SDK 获取 id_token
        ↓
前端将 id_token 发送到后端 API
        ↓
后端验证 token → 返回 JWT + 用户信息
```

## 1. 引入 Google Identity Services (GIS)

```html
<script src="https://accounts.google.com/gsi/client" async></script>
```

## 2. 初始化并触发登录

```javascript
const GOOGLE_CLIENT_ID = "358687166908-pp8q17uu638a4boh8lptvhv34q94qp2a.apps.googleusercontent.com";

// 初始化
google.accounts.id.initialize({
  client_id: GOOGLE_CLIENT_ID,
  callback: handleGoogleLogin
});

// 渲染 Google 登录按钮
google.accounts.id.renderButton(
  document.getElementById("google-signin-btn"),
  { theme: "outline", size: "large", text: "signin_with", locale: "zh-TW" }
);

// 回调：拿到 id_token 后调用后端接口
async function handleGoogleLogin(response) {
  const res = await fetch("https://52.195.4.10:8080/api/v1/auth/google", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id_token: response.credential })
  });
  const data = await res.json();

  if (data.code === 0) {
    // 登录成功
    // data.data.token    → JWT token，后续请求放 Authorization header
    // data.data.user_id  → 用户 ID
    // data.data.email    → 邮箱
    // data.data.nickname → 昵称
    localStorage.setItem("token", data.data.token);
  } else {
    // 登录失败
    console.error("Google login failed:", data.message);
  }
}
```

## 3. 后端 API

### POST /api/v1/auth/google

**Request:**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (成功):**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 42,
    "email": "user@gmail.com",
    "nickname": "张三"
  }
}
```

**Response (失败):**
```json
{
  "code": 40007,
  "message": "invalid OAuth token"
}
```

## 4. 登录后使用 JWT

所有需要认证的接口，在请求头加上：

```
Authorization: Bearer <token>
```

示例：
```javascript
fetch("/api/v1/cart", {
  headers: {
    "Authorization": "Bearer " + localStorage.getItem("token")
  }
});
```

## 5. 后端行为说明

| 场景 | 后端处理 |
|------|----------|
| 新用户首次 Google 登录 | 自动创建账号，昵称取 Google 名称 |
| 已注册邮箱的用户 Google 登录 | 自动关联，无需重新注册 |
| 同一用户再次 Google 登录 | 直接返回 token |

## 6. 错误码

| code | 含义 |
|------|------|
| 0 | 成功 |
| 40002 | 请求体格式错误 / 缺少 id_token |
| 40007 | Google token 无效或过期 |
| 40008 | OAuth 服务端错误 |

## 注意事项

- `client_id` 前后端必须使用同一个，否则 token 验证会失败
- Google SDK 会自动处理授权弹窗，前端不需要手动拼接 OAuth URL
- JWT 有效期 72 小时，过期后需重新登录
