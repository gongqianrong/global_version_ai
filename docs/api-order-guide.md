# 订单系统 API 文档

Base URL: `http://52.195.4.10:8080/api/v1`

所有接口需要认证：`Authorization: Bearer <token>`

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

## 下单流程

```
预订单预览 → 生成订单 → 提交支付
(settlement)   (confirm)    (pay)
```

1. **预订单**：前端展示价格明细，用户确认
2. **生成订单**：创建订单记录（待支付状态），从购物车移除商品
3. **提交支付**：扣除钱包余额，订单变为已支付

---

## 1. 预订单（结算预览）

商品价格预览，不写入数据库。返回商品明细、费用计算、钱包余额。

```
POST /order/settlement
```

**请求体：**
```json
{
  "items": [
    { "product_id": "surugaya_185008572", "quantity": 1 },
    { "product_id": "surugaya_185009001", "quantity": 2 }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | array | 是 | 商品列表，至少 1 项 |
| items[].product_id | string | 是 | 商品 ID |
| items[].quantity | int | 否 | 数量，默认 1 |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "orderTotalJp": 23500,
    "commissionFeeJp": 2000,
    "orderInpriceJp": 23500,
    "orderRate": 1,
    "orderPaytype": 0,
    "orderType": 0,
    "isChange": 0,
    "totalShippingFee": 1500,
    "walletBalance": 105600,
    "orderDetailList": [
      {
        "goodsMid": "surugaya_185008572",
        "goodsName": "フィギュア xxx",
        "goodsNum": 1,
        "goodsImg": "https://www.suruga-ya.jp/pics/boxart_m/...",
        "goodsUrl": "https://www.suruga-ya.jp/product/detail/...",
        "goodsSellerId": "Shop A",
        "goodsAmountJp": 10000,
        "commissionFeeJp": 1000,
        "shippingFee": 500,
        "platform": "surugaya",
        "condition": "used",
        "sellerId": "seller_001",
        "state": 0,
        "blockLevel": 0
      }
    ]
  }
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| orderTotalJp | int64 | 订单总价（JPY）= 商品价 + 手续费 + 运费 |
| commissionFeeJp | int64 | 代购手续费总计（商品价 × 10%） |
| orderInpriceJp | int64 | 实际应付金额 |
| totalShippingFee | int64 | 运费总计 |
| walletBalance | int64 | 用户当前钱包余额 |
| orderDetailList | array | 商品明细列表 |

**商品明细字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| goodsMid | string | 商品 ID |
| goodsName | string | 商品名称 |
| goodsNum | int | 数量 |
| goodsImg | string | 商品图片 URL |
| goodsUrl | string | 商品源站链接 |
| goodsSellerId | string | 卖家名称 |
| goodsAmountJp | int64 | 单件商品价格（JPY） |
| commissionFeeJp | int64 | 单件手续费 |
| shippingFee | int64 | 单件运费 |
| platform | string | 平台：surugaya / yahoo_auction |
| condition | string | 商品成色：new / like_new / good / fair / poor |
| sellerId | string | 卖家 ID |
| state | int | 0=正常, 1=商品不可用 |
| blockLevel | int | 保留字段，暂为 0 |

---

## 2. 生成订单

创建订单（待支付状态），并从购物车移除已下单商品。**不扣款。**

```
POST /order/confirm
```

**请求体：**
```json
{
  "items": [
    { "product_id": "surugaya_185008572", "quantity": 1 }
  ],
  "order_paytype": 0,
  "order_remark": "请帮忙检查外包装",
  "order_purchase_type": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | array | 是 | 同预订单 |
| order_paytype | int | 否 | 支付方式，默认 0（钱包余额） |
| order_remark | string | 否 | 订单备注 |
| order_purchase_type | int | 否 | 代购类型，默认 1 |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "orderNumber": "RO20260304150000A1B2",
    "orderState": 0,
    "orderTotalJp": 11500,
    "commissionFeeJp": 1000,
    "orderInpriceJp": 11500,
    "orderPaytype": 0,
    "totalShippingFee": 500,
    "orderDetailList": [ ... ]
  }
}
```

| 字段 | 说明 |
|------|------|
| orderNumber | 订单号（后续支付/查询/取消用） |
| orderState | 0 = 待支付 |

---

## 3. 提交支付

对待支付订单扣除钱包余额，订单状态变为已支付。

```
POST /order/pay
```

**请求体：**
```json
{
  "order_number": "RO20260304150000A1B2"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| order_number | string | 是 | 订单号 |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "orderNumber": "RO20260304150000A1B2",
    "orderState": 1,
    "paidAmount": 11500,
    "balanceAfter": 88500
  }
}
```

| 字段 | 说明 |
|------|------|
| orderState | 1 = 已支付 |
| paidAmount | 本次扣款金额（JPY） |
| balanceAfter | 扣款后钱包余额 |

---

## 4. 订单详情

```
GET /order/{orderNumber}
```

**示例：** `GET /order/RO20260304150000A1B2`

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "orderNumber": "RO20260304150000A1B2",
    "orderState": 1,
    "orderTotalJp": 11500,
    "commissionFeeJp": 1000,
    "shippingFeeJp": 500,
    "orderInpriceJp": 11500,
    "orderPaytype": 0,
    "orderRemark": "请帮忙检查外包装",
    "createdAt": "2026-03-04 15:00:00",
    "orderDetailList": [
      {
        "goodsMid": "surugaya_185008572",
        "goodsName": "フィギュア xxx",
        "goodsNum": 1,
        "goodsImg": "https://...",
        "goodsUrl": "https://...",
        "goodsAmountJp": 10000,
        "commissionFeeJp": 1000,
        "shippingFee": 500,
        "sellerId": "seller_001",
        "sellerName": "Shop A",
        "platform": "surugaya",
        "condition": "used",
        "state": 0
      }
    ]
  }
}
```

---

## 5. 订单列表

```
GET /orders?page=1&page_size=20&state=-1
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20，最大 100 |
| state | int | 否 | 按状态筛选，默认 -1（全部） |

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "orderNumber": "RO20260304150000A1B2",
        "orderState": 1,
        "orderTotalJp": 11500,
        "commissionFeeJp": 1000,
        "shippingFeeJp": 500,
        "orderInpriceJp": 11500,
        "orderPaytype": 0,
        "orderRemark": "",
        "createdAt": "2026-03-04 15:00:00"
      }
    ],
    "total": 15,
    "page": 1
  }
}
```

> 注：列表接口不返回 `orderDetailList`，需查看明细请调用订单详情接口。

---

## 6. 取消订单

仅**待支付**（state=0）的订单可以取消。

```
POST /order/cancel
```

**请求体：**
```json
{
  "order_number": "RO20260304150000A1B2"
}
```

**成功响应：**
```json
{
  "code": 0,
  "data": {
    "orderNumber": "RO20260304150000A1B2",
    "orderState": 8
  }
}
```

---

## 订单状态码

| state | 说明 |
|-------|------|
| 0 | 待支付 |
| 1 | 已支付（待代购） |
| 2 | 代购中 |
| 3 | 已入库 |
| 4 | 打包中 |
| 5 | 已打包 |
| 6 | 已发货 |
| 7 | 已完成 |
| 8 | 已取消 |
| 9 | 已退款 |

> 当前版本前端可见状态：0（待支付）、1（已支付）、8（已取消）。其余状态由运营后台流转。

---

## 费用计算规则

| 费用项 | 计算方式 |
|--------|----------|
| 商品价格 | 从 ES 缓存获取日本站实时价格 |
| 代购手续费 | 商品价格 × 10% |
| 运费 | 从商品数据 shipping_fee_jpy 字段获取 |
| 订单总价 | Σ (商品价格 + 手续费 + 运费) × 数量 |

---

## 错误码

| code | HTTP | 说明 |
|------|------|------|
| 40002 | 400 | 请求体格式错误 / 缺少必要参数 |
| 40010 | 400 | 商品不存在或已下架 |
| 40011 | 400 | 钱包余额不足 |
| 40012 | 400 | 订单不可支付（非待支付状态） |
| 40013 | 400 | 订单不可取消（非待支付状态） |
| 40100 | 401 | 未提供认证信息 |
| 40101 | 401 | Token 无效或已过期 |
| 40401 | 404 | 订单不存在 |
| 50001 | 500 | 服务器内部错误 |

---

## 前端接入示例

### 完整下单流程

```javascript
const BASE = 'http://52.195.4.10:8080/api/v1';
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${token}`
};

// Step 1: 预订单 — 展示价格确认页
const settlement = await fetch(`${BASE}/order/settlement`, {
  method: 'POST', headers,
  body: JSON.stringify({
    items: [{ product_id: 'surugaya_185008572', quantity: 1 }]
  })
}).then(r => r.json());

// 展示 settlement.data.orderTotalJp / walletBalance 等信息
// 用户点击「确认下单」

// Step 2: 生成订单
const order = await fetch(`${BASE}/order/confirm`, {
  method: 'POST', headers,
  body: JSON.stringify({
    items: [{ product_id: 'surugaya_185008572', quantity: 1 }],
    order_remark: '备注内容'
  })
}).then(r => r.json());

const orderNumber = order.data.orderNumber;
// 跳转到支付页，展示订单信息

// Step 3: 用户点击「支付」
const pay = await fetch(`${BASE}/order/pay`, {
  method: 'POST', headers,
  body: JSON.stringify({ order_number: orderNumber })
}).then(r => r.json());

if (pay.code === 0) {
  // 支付成功，跳转到订单详情或成功页
  console.log('支付成功，剩余余额:', pay.data.balanceAfter);
} else if (pay.code === 40011) {
  // 余额不足，引导充值
}
```

### 订单列表 + 详情

```javascript
// 获取全部订单
const list = await fetch(`${BASE}/orders?page=1&page_size=20`, {
  headers
}).then(r => r.json());

// 按状态筛选（如只看待支付）
const pending = await fetch(`${BASE}/orders?state=0`, {
  headers
}).then(r => r.json());

// 查看订单详情
const detail = await fetch(`${BASE}/order/${orderNumber}`, {
  headers
}).then(r => r.json());
```

### 取消未支付订单

```javascript
const cancel = await fetch(`${BASE}/order/cancel`, {
  method: 'POST', headers,
  body: JSON.stringify({ order_number: orderNumber })
}).then(r => r.json());

if (cancel.code === 40013) {
  alert('该订单已支付，无法取消');
}
```
