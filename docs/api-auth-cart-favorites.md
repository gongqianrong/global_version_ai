# API 认证 / 购物车 / 收藏夹 — 前端接入文档

> Base URL: `http://52.195.4.10:8080`
> 所有响应格式均为标准信封：`{ "code": 0, "data": {...}, "message": "...", "request_id": "..." }`
> 多语言：请求头 `Accept-Language: zh-TW | ja | en`（与现有接口一致）

---

## 一、认证（公开接口，无需 Token）

### 1.1 注册 — `POST /api/v1/auth/register`

**请求体：**
```json
{
  "email": "user@example.com",
  "password": "mypassword",
  "nickname": "小明"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `email` | string | 是 | 邮箱地址，唯一 |
| `password` | string | 是 | 密码，最少 6 位 |
| `nickname` | string | 否 | 昵称，默认空字符串 |

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "email": "user@example.com",
    "nickname": "小明"
  },
  "request_id": "a1b2c3d4e5f6"
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 400 | 40002 | 请求体 JSON 解析失败 |
| 400 | 40003 | email/password 为空，或密码少于 6 位 |
| 409 | 40901 | 邮箱已被注册 |

---

### 1.2 登录 — `POST /api/v1/auth/login`

**请求体：**
```json
{
  "email": "user@example.com",
  "password": "mypassword"
}
```

**成功响应 `200`：**
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

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 400 | 40002 | 请求体 JSON 解析失败 |
| 400 | 40003 | email/password 为空 |
| 401 | 40100 | 邮箱不存在或密码错误 |

---

### 1.3 Token 使用说明

- 注册/登录成功后返回 `token` 字段，为 JWT（HS256），**有效期 72 小时**
- 购物车和收藏夹接口必须在请求头中携带：
  ```
  Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
  ```
- Token 过期或无效时，所有受保护接口返回 `401` + `code: 40101`

**前端建议：**

```javascript
// Axios — 登录后全局设置
const { data } = await axios.post('/api/v1/auth/login', { email, password });
const token = data.data.token;
axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;

// AsyncStorage 持久化（React Native）
await AsyncStorage.setItem('token', token);
```

---

## 二、购物车（需认证）

> 所有购物车接口需要 `Authorization: Bearer <token>` 请求头

### 2.1 查看购物车 — `GET /api/v1/cart`

按卖家分组返回，包含实时商品信息（价格、翻译标题、状态）和价格变动检测。

**请求示例：**
```
GET /api/v1/cart
Authorization: Bearer <token>
Accept-Language: zh-TW
```

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "groups": [
      {
        "seller_id": "shop_12345",
        "seller_name": "駿河屋 本店",
        "items": [
          {
            "product_id": "surugaya_663043159",
            "quantity": 2,
            "price_at_add": 3500,
            "current_price": 3800,
            "price_changed": true,
            "title": "高達模型 RX-78-2（翻譯後）",
            "title_original": "ガンダム RX-78-2 プラモデル",
            "image": "https://...",
            "platform": "surugaya",
            "status": "available",
            "seller_id": "shop_12345",
            "seller_name": "駿河屋 本店",
            "is_translated": true
          }
        ]
      }
    ],
    "total_items": 2
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `groups` | array | 按卖家分组的购物车列表 |
| `groups[].seller_id` | string | 卖家 ID |
| `groups[].seller_name` | string | 卖家名称 |
| `groups[].items` | array | 该卖家下的商品列表 |
| `product_id` | string | 商品统一 ID（如 `surugaya_663043159`） |
| `quantity` | int | 数量 |
| `price_at_add` | int64 | 加入购物车时的价格（日元） |
| `current_price` | int64 | 当前实时价格（日元） |
| `price_changed` | bool | **价格是否变动**（前端可用于提示用户） |
| `title` | string | 商品标题（根据 Accept-Language 翻译） |
| `title_original` | string | 日文原始标题 |
| `image` | string | 商品首图 URL |
| `platform` | string | 来源平台 |
| `status` | string | 商品状态（`available` / `sold` / `reserved` / `delisted`） |
| `is_translated` | bool | 标题是否已翻译 |
| `total_items` | int | 购物车总商品件数（含数量） |

---

### 2.2 加入购物车 — `POST /api/v1/cart`

**请求体：**
```json
{
  "product_id": "surugaya_663043159",
  "quantity": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `product_id` | string | 是 | 商品 ID |
| `quantity` | int | 否 | 数量，默认 1。若商品已在购物车中，数量**累加** |

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "user_id": 1,
    "product_id": "surugaya_663043159",
    "quantity": 1,
    "price_at_add": 3500,
    "added_at": "2026-03-03T10:00:00Z",
    "updated_at": "2026-03-03T10:00:00Z"
  }
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 400 | 40002 | product_id 为空 |
| 400 | 40003 | 商品不可购买（已下架/已售出） |
| 404 | 40401 | 商品不存在 |

> **行为说明**：同一商品重复加入时，数量会累加（不会报错），`price_at_add` 保持首次加入时的价格。

---

### 2.3 更新数量 — `PUT /api/v1/cart/{productID}`

**请求体：**
```json
{
  "quantity": 3
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `quantity` | int | 是 | 新数量，必须 > 0 |

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "product_id": "surugaya_663043159",
    "quantity": 3
  }
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 400 | 40003 | quantity <= 0 |
| 404 | 40401 | 购物车中无此商品 |

---

### 2.4 移除商品 — `DELETE /api/v1/cart/{productID}`

**请求示例：**
```
DELETE /api/v1/cart/surugaya_663043159
Authorization: Bearer <token>
```

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "product_id": "surugaya_663043159",
    "status": "removed"
  }
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 404 | 40401 | 购物车中无此商品 |

---

## 三、收藏夹（需认证）

> 所有收藏夹接口需要 `Authorization: Bearer <token>` 请求头

### 3.1 收藏列表 — `GET /api/v1/favorites`

支持分页，返回收藏商品的实时信息（翻译标题、价格、状态）。

**查询参数：**

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | 1 | 页码 |
| `page_size` | int | 20 | 每页条数，最大 100 |

**请求示例：**
```
GET /api/v1/favorites?page=1&page_size=20
Authorization: Bearer <token>
Accept-Language: en
```

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "product_id": "surugaya_663043159",
        "title": "Gundam RX-78-2 Plastic Model",
        "title_original": "ガンダム RX-78-2 プラモデル",
        "image": "https://...",
        "price_jpy": 3500,
        "platform": "surugaya",
        "status": "available",
        "is_translated": true,
        "added_at": "2026-03-03T10:00:00Z"
      }
    ],
    "total": 42,
    "page": 1
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `items` | array | 收藏商品列表 |
| `product_id` | string | 商品 ID |
| `title` | string | 翻译后标题（根据 Accept-Language） |
| `title_original` | string | 日文原始标题 |
| `image` | string | 商品首图 |
| `price_jpy` | int64 | 当前价格（日元） |
| `platform` | string | 来源平台 |
| `status` | string | 商品状态 |
| `is_translated` | bool | 标题是否已翻译 |
| `added_at` | string | 收藏时间（ISO 8601） |
| `total` | int64 | 收藏总数 |
| `page` | int | 当前页码 |

---

### 3.2 添加收藏 — `POST /api/v1/favorites/{productID}`

**请求示例：**
```
POST /api/v1/favorites/surugaya_663043159
Authorization: Bearer <token>
```

无请求体。

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "user_id": 1,
    "product_id": "surugaya_663043159",
    "added_at": "2026-03-03T10:00:00Z"
  }
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 404 | 40401 | 商品不存在 |

> **行为说明**：重复收藏同一商品不会报错，返回已有记录（幂等操作）。

---

### 3.3 取消收藏 — `DELETE /api/v1/favorites/{productID}`

**请求示例：**
```
DELETE /api/v1/favorites/surugaya_663043159
Authorization: Bearer <token>
```

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "product_id": "surugaya_663043159",
    "status": "removed"
  }
}
```

**错误：**
| HTTP | code | 场景 |
|------|------|------|
| 404 | 40401 | 未收藏此商品 |

---

### 3.4 批量检查收藏状态 — `GET /api/v1/favorites/check`

用于列表页渲染心形图标（是否已收藏）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `product_ids` | string | 是 | 逗号分隔的商品 ID 列表 |

**请求示例：**
```
GET /api/v1/favorites/check?product_ids=surugaya_663043159,surugaya_123456,yahoo_auction_abc
Authorization: Bearer <token>
```

**成功响应 `200`：**
```json
{
  "code": 0,
  "data": {
    "surugaya_663043159": true,
    "surugaya_123456": false,
    "yahoo_auction_abc": true
  }
}
```

> 返回的 map 中，`true` 表示已收藏，`false` 表示未收藏。所有传入的 ID 都会出现在结果中。

---

## 四、错误码汇总

| code | HTTP | 含义（zh-TW） | 含义（en） |
|------|------|--------------|-----------|
| 40002 | 400 | 缺少必要參數 | missing required parameter |
| 40003 | 400 | 參數值無效 | invalid parameter value |
| 40100 | 401 | 需要認證 | authentication required |
| 40101 | 401 | 無效或過期的令牌 | invalid or expired token |
| 40401 | 404 | 找不到商品 | product not found |
| 40901 | 409 | 電子郵件已被註冊 | email already registered |
| 50001 | 500 | 內部伺服器錯誤 | internal server error |

> 错误消息会根据 `Accept-Language` 自动翻译，与现有接口行为一致。

---

## 五、前端接入示例

### React Native / Axios 完整流程

```javascript
import axios from 'axios';
import AsyncStorage from '@react-native-async-storage/async-storage';

const api = axios.create({
  baseURL: 'http://52.195.4.10:8080/api/v1',
  headers: { 'Accept-Language': 'zh-TW' },
});

// ---- Token 拦截器 ----
api.interceptors.request.use(async (config) => {
  const token = await AsyncStorage.getItem('token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    if (err.response?.data?.code === 40101) {
      // Token 过期，跳转登录页
      await AsyncStorage.removeItem('token');
      // navigation.navigate('Login');
    }
    return Promise.reject(err);
  }
);

// ---- 认证 ----
async function register(email, password, nickname) {
  const { data } = await api.post('/auth/register', { email, password, nickname });
  await AsyncStorage.setItem('token', data.data.token);
  return data.data;
}

async function login(email, password) {
  const { data } = await api.post('/auth/login', { email, password });
  await AsyncStorage.setItem('token', data.data.token);
  return data.data;
}

// ---- 购物车 ----
async function getCart() {
  const { data } = await api.get('/cart');
  return data.data; // { groups: [...], total_items: N }
}

async function addToCart(productId, quantity = 1) {
  const { data } = await api.post('/cart', { product_id: productId, quantity });
  return data.data;
}

async function updateCartQuantity(productId, quantity) {
  const { data } = await api.put(`/cart/${productId}`, { quantity });
  return data.data;
}

async function removeFromCart(productId) {
  const { data } = await api.delete(`/cart/${productId}`);
  return data.data;
}

// ---- 收藏夹 ----
async function getFavorites(page = 1, pageSize = 20) {
  const { data } = await api.get(`/favorites?page=${page}&page_size=${pageSize}`);
  return data.data; // { items: [...], total: N, page: N }
}

async function addFavorite(productId) {
  const { data } = await api.post(`/favorites/${productId}`);
  return data.data;
}

async function removeFavorite(productId) {
  const { data } = await api.delete(`/favorites/${productId}`);
  return data.data;
}

async function checkFavorites(productIds) {
  const ids = productIds.join(',');
  const { data } = await api.get(`/favorites/check?product_ids=${ids}`);
  return data.data; // { "id1": true, "id2": false }
}
```

### 购物车页面 — 价格变动提示

```jsx
{item.price_changed && (
  <Text style={{ color: 'red' }}>
    价格已变动: ¥{item.price_at_add} → ¥{item.current_price}
  </Text>
)}
```

### 搜索列表 — 收藏心形图标

```javascript
// 搜索结果加载后，批量检查收藏状态
const productIds = searchResults.map(p => p.id);
const favMap = await checkFavorites(productIds);

// 渲染
{searchResults.map(product => (
  <HeartIcon
    filled={favMap[product.id]}
    onPress={() => favMap[product.id]
      ? removeFavorite(product.id)
      : addFavorite(product.id)
    }
  />
))}
```

---

## 六、cURL 测试命令

```bash
# 注册
curl -X POST http://52.195.4.10:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"12345678","nickname":"test"}'

# 登录
curl -X POST http://52.195.4.10:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"12345678"}'

# 设置 Token 变量（将返回的 token 值替换到下面）
TOKEN="eyJhbGciOiJIUzI1NiIs..."

# 查看购物车
curl -H "Authorization: Bearer $TOKEN" \
     -H "Accept-Language: zh-TW" \
     http://52.195.4.10:8080/api/v1/cart

# 加入购物车
curl -X POST http://52.195.4.10:8080/api/v1/cart \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id":"surugaya_663043159","quantity":1}'

# 更新购物车数量
curl -X PUT http://52.195.4.10:8080/api/v1/cart/surugaya_663043159 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity":3}'

# 移除购物车商品
curl -X DELETE http://52.195.4.10:8080/api/v1/cart/surugaya_663043159 \
  -H "Authorization: Bearer $TOKEN"

# 收藏列表
curl -H "Authorization: Bearer $TOKEN" \
     -H "Accept-Language: en" \
     "http://52.195.4.10:8080/api/v1/favorites?page=1&page_size=20"

# 添加收藏
curl -X POST http://52.195.4.10:8080/api/v1/favorites/surugaya_663043159 \
  -H "Authorization: Bearer $TOKEN"

# 取消收藏
curl -X DELETE http://52.195.4.10:8080/api/v1/favorites/surugaya_663043159 \
  -H "Authorization: Bearer $TOKEN"

# 批量检查收藏状态
curl -H "Authorization: Bearer $TOKEN" \
     "http://52.195.4.10:8080/api/v1/favorites/check?product_ids=surugaya_663043159,surugaya_123456"
```
