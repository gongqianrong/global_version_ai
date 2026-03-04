# 钱包 API 文档

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

所有钱包接口均需要 JWT 认证：
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

---

## 1. 查询余额

```
GET /wallet/balance
```

**请求头：**
```
Authorization: Bearer <token>
```

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "balance": 105600,
    "currency": "JPY"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| balance | number | 余额，单位日元（整数） |
| currency | string | 货币类型，固定 `"JPY"` |

**说明：**
- 用户首次调用时会自动创建钱包，初始余额为 0
- 余额单位为日元，整数，无小数

---

## 2. 查询流水记录

```
GET /wallet/transactions?page=1&page_size=20
```

**请求头：**
```
Authorization: Bearer <token>
```

**Query 参数：**

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| page | 否 | 1 | 页码，从 1 开始 |
| page_size | 否 | 20 | 每页条数，最大 100 |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 3,
        "user_id": 1,
        "type": "purchase",
        "amount": -3000,
        "balance_after": 7000,
        "description": "订单 ORD-20260304-001",
        "related_order": "ORD-20260304-001",
        "created_at": "2026-03-04T15:30:00Z"
      },
      {
        "id": 2,
        "user_id": 1,
        "type": "recharge",
        "amount": 10000,
        "balance_after": 10000,
        "description": "管理员充值",
        "created_at": "2026-03-04T10:00:00Z"
      }
    ],
    "total": 2,
    "page": 1
  }
}
```

**流水字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 流水 ID |
| user_id | number | 用户 ID |
| type | string | 交易类型（见下表） |
| amount | number | 交易金额，正数=入账，负数=出账 |
| balance_after | number | 交易后余额快照 |
| description | string | 交易描述 |
| related_order | string \| null | 关联订单号（无则为 null） |
| created_at | string | 交易时间 (ISO 8601) |

**交易类型 `type`：**

| 值 | 说明 | amount 方向 |
|------|------|-------------|
| recharge | 充值 | 正数 |
| purchase | 消费（下单扣款） | 负数 |
| refund | 退款 | 正数 |
| adjustment | 管理员调账 | 正数或负数 |

**排序：** 按 `created_at` 降序（最新的在前面）。

---

## 3. 管理员手动充值/调账

> 此接口仅管理员可用，需要 JWT 中的用户具有 admin 权限。

```
POST /admin/wallet/adjust
```

**请求头：**
```
Authorization: Bearer <admin_token>
```

**请求体：**
```json
{
  "user_id": 5,
  "amount": 10000,
  "type": "recharge",
  "description": "首充赠送"
}
```

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| user_id | 是 | number | 目标用户 ID |
| amount | 是 | number | 金额，正数=入账，负数=出账，不能为 0 |
| type | 是 | string | 交易类型：`recharge`、`adjustment`、`refund` |
| description | 否 | string | 操作描述 |
| related_order | 否 | string | 关联订单号 |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "balance": 105600,
    "transaction_id": 12
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| balance | number | 操作后的最新余额 |
| transaction_id | number | 本次操作生成的流水 ID |

**使用示例：**

充值 10000 日元：
```json
{ "user_id": 5, "amount": 10000, "type": "recharge", "description": "管理员充值" }
```

扣除 3000 日元：
```json
{ "user_id": 5, "amount": -3000, "type": "adjustment", "description": "调账扣除" }
```

退款 5000 日元：
```json
{ "user_id": 5, "amount": 5000, "type": "refund", "description": "订单退款", "related_order": "ORD-001" }
```

---

## 前端接入示例

### 钱包页面 — 查询余额 + 流水

```javascript
const API_BASE = 'http://52.195.4.10:8080/api/v1';
const token = localStorage.getItem('token');

// 查余额
async function getBalance() {
  const res = await fetch(`${API_BASE}/wallet/balance`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const data = await res.json();
  if (data.code === 0) {
    console.log(`余额: ¥${data.data.balance}`);
  }
}

// 查流水（分页）
async function getTransactions(page = 1) {
  const res = await fetch(`${API_BASE}/wallet/transactions?page=${page}&page_size=20`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const data = await res.json();
  if (data.code === 0) {
    const { items, total, page } = data.data;
    items.forEach(tx => {
      const sign = tx.amount > 0 ? '+' : '';
      console.log(`${tx.type} ${sign}¥${tx.amount} → 余额 ¥${tx.balance_after}`);
    });
  }
}
```

### 管理后台 — 充值操作

```javascript
async function adjustWallet(userId, amount, type, description) {
  const res = await fetch(`${API_BASE}/admin/wallet/adjust`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${adminToken}`
    },
    body: JSON.stringify({
      user_id: userId,
      amount: amount,
      type: type,
      description: description
    })
  });
  const data = await res.json();
  if (data.code === 0) {
    console.log(`操作成功，新余额: ¥${data.data.balance}，流水ID: ${data.data.transaction_id}`);
  } else {
    console.error('操作失败:', data.message);
  }
}

// 充值
adjustWallet(5, 10000, 'recharge', '管理员充值');

// 调账扣除
adjustWallet(5, -3000, 'adjustment', '调账扣除');
```

---

## 错误码

| code | HTTP 状态码 | 说明 |
|------|------------|------|
| 40002 | 400 | 请求体格式错误 / 缺少必要参数 / amount 为 0 / type 无效 |
| 40009 | 400 | 余额不足（扣款时余额不够） |
| 40100 | 401 | 未提供认证信息 |
| 40101 | 401 | Token 无效或已过期 |
| 40300 | 403 | 需要管理员权限（非 admin 用户调用管理接口） |
| 50001 | 500 | 服务器内部错误 |

---

## 接口总览

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/wallet/balance` | JWT | 查询当前用户余额 |
| GET | `/wallet/transactions` | JWT | 查询当前用户流水记录（分页） |
| POST | `/admin/wallet/adjust` | JWT + Admin | 管理员手动充值/调账/退款 |
