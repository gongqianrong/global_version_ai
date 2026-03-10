# 用户认证 API 文档

Base URL: `http://52.195.4.10:8080/api/v1`

所有响应格式统一：
```json
{
  "code": 0,
  "data": { ... },
  "message": "",
  "request_id": "xxxxx"
}
```
`code` 为 0 表示成功，非 0 为错误码。

---

## 1. 发送验证码

注册前必须先获取邮箱验证码。

```
POST /auth/send-code
```

**请求体：**
```json
{
  "email": "user@example.com"
}
```

**成功响应：**
```json
{
  "code": 0,
  "data": { "message": "verification code sent" }
}
```

**限制：**
- 同一邮箱 60 秒内只能发送一次
- 验证码 10 分钟有效

**错误码：**
| code | 说明 |
|------|------|
| 40004 | 邮箱格式无效 |
| 40005 | 发送过于频繁，请稍后再试 |

---

## 2. 邮箱注册

```
POST /auth/register
```

**请求体：**
```json
{
  "email": "user@example.com",
  "password": "123456",
  "nickname": "小明",
  "code": "382951"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| email | 是 | 邮箱地址 |
| password | 是 | 密码，最少 6 位 |
| nickname | 否 | 昵称，默认空字符串 |
| code | 是 | 邮箱验证码（通过 send-code 获取） |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "email": "user@example.com",
    "nickname": "小明"
  }
}
```

**错误码：**
| code | 说明 |
|------|------|
| 40003 | 缺少 email/password 或密码太短 |
| 40006 | 验证码无效或已过期 |
| 40901 | 邮箱已被注册 |

---

## 3. 邮箱密码登录

```
POST /auth/login
```

**请求体：**
```json
{
  "email": "user@example.com",
  "password": "123456"
}
```

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "email": "user@example.com",
    "nickname": "小明"
  }
}
```

**错误码：**
| code | 说明 |
|------|------|
| 40100 | 邮箱或密码错误 |

---

## 4. Google 登录

前端使用 Google Sign-In SDK 获取 `id_token`，发送给后端验证。

```
POST /auth/google
```

**请求体：**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "email": "user@gmail.com",
    "nickname": "John Doe"
  }
}
```

**账号关联逻辑：**
- Google 邮箱匹配已有用户 → 自动关联，返回该用户的 JWT
- 无匹配 → 自动创建新用户（密码登录不可用）
- 重复登录 → 返回同一个 user_id

**错误码：**
| code | 说明 |
|------|------|
| 40002 | 请求体格式错误或缺少 id_token |
| 40007 | id_token 无效（签名错误、过期、audience 不匹配） |
| 40008 | OAuth 服务异常 |

### 前端接入示例

```html
<!-- 引入 Google SDK -->
<script src="https://accounts.google.com/gsi/client" async></script>

<!-- 登录按钮 -->
<div id="g_id_onload"
     data-client_id="358687166908-pp8q17uu638a4boh8lptvhv34q94qp2a.apps.googleusercontent.com"
     data-callback="handleGoogleLogin">
</div>
<div class="g_id_signin" data-type="standard"></div>

<script>
function handleGoogleLogin(response) {
  fetch('http://52.195.4.10:8080/api/v1/auth/google', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id_token: response.credential })
  })
  .then(res => res.json())
  .then(data => {
    if (data.code === 0) {
      localStorage.setItem('token', data.data.token);
      console.log('登录成功', data.data);
    } else {
      console.error('登录失败', data.message);
    }
  });
}
</script>
```

---

## 5. Apple 登录（暂未启用）

预留接口，待移动端 App 上线后启用。

```
POST /auth/apple
```

**请求体：**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

响应格式与 Google 登录相同。

---

## JWT Token 使用

登录成功后返回的 `token` 需要在后续需要认证的请求中携带：

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Token 有效期：72 小时**

### 需要认证的接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /cart | 获取购物车 |
| POST | /cart | 添加商品到购物车 |
| PUT | /cart/{productID} | 修改购物车商品数量 |
| DELETE | /cart/{productID} | 删除购物车商品 |
| GET | /favorites | 获取收藏列表 |
| POST | /favorites/{productID} | 收藏商品 |
| DELETE | /favorites/{productID} | 取消收藏 |
| GET | /favorites/check?ids=a,b,c | 批量检查是否已收藏 |

**未携带或 Token 过期时的错误：**
| code | 说明 |
|------|------|
| 40100 | 未提供认证信息 |
| 40101 | Token 无效或已过期 |

---

## 完整错误码表

| code | 说明 |
|------|------|
| 40002 | 请求体格式错误 / 缺少必要参数 |
| 40003 | 参数值无效 |
| 40004 | 邮箱格式无效 |
| 40005 | 验证码发送过于频繁 |
| 40006 | 验证码无效或已过期 |
| 40007 | OAuth token 无效 |
| 40008 | OAuth 服务异常 |
| 40100 | 需要认证 / 凭证错误 |
| 40101 | Token 无效或已过期 |
| 40901 | 邮箱已被注册 |
| 50001 | 服务器内部错误 |
