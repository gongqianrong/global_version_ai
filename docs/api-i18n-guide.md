# API 多语言调用指南

## 使用方式

所有接口通过 HTTP 请求头 `Accept-Language` 切换语言，无需额外参数。

```
Accept-Language: zh-TW   # 繁体中文（默认）
Accept-Language: ja       # 日文
Accept-Language: en       # 英文
```

**不传此头部时默认返回繁体中文。**

---

## 前端接入示例

### Axios 全局配置
```javascript
axios.defaults.headers.common['Accept-Language'] = 'zh-TW';
```

### Fetch 单次请求
```javascript
fetch('http://52.195.4.10:8080/api/v1/products/surugaya_663043159', {
  headers: { 'Accept-Language': 'en' }
})
```

### React Native 切换语言
```javascript
const lang = i18n.locale; // "zh-TW" | "ja" | "en"
api.defaults.headers['Accept-Language'] = lang;
```

---

## 受影响的字段

### 1. 商品详情 `GET /api/v1/products/{id}`

| 字段 | zh-TW | ja | en |
|------|-------|----|----|
| `status` | 已售出 / 可購買 / 已預約 / 已下架 | 売り切れ / 販売中 / 取り置き中 / 掲載終了 | Sold / Available / Reserved / Delisted |
| `condition` | 全新 / 近全新 / 良好 / 尚可 / 較差 | 新品 / 未使用に近い / 良好 / やや傷あり / 傷あり | New / Like New / Good / Fair / Poor |
| `shipping_type` | 免運費 / 買家自付 / 含運費 | 送料無料 / 送料購入者負担 / 送料込み | Free Shipping / Buyer Pays / Included |
| `content_rating` | 一般 / R18 | 一般 / R18 | General / R18 |
| `title` | 翻译后标题（如有） | 日文原文 | 翻译后标题（如有） |
| `description` | 翻译后描述（如有） | 日文原文 | 翻译后描述（如有） |

> `title_original` 和 `description_original` 始终返回日文原文，不受语言影响。
> `is_translated` 为 `true` 时表示 `title` 已被翻译，为 `false` 时 `title` 是日文原文。

**示例响应对比：**

```
GET /api/v1/products/surugaya_663043159
Accept-Language: zh-TW
```
```json
{
  "code": 0,
  "data": {
    "status": "已售出",
    "content_rating": "一般",
    "is_translated": true
  }
}
```

```
Accept-Language: en
```
```json
{
  "code": 0,
  "data": {
    "status": "Sold",
    "content_rating": "General",
    "is_translated": true
  }
}
```

```
Accept-Language: ja
```
```json
{
  "code": 0,
  "data": {
    "status": "売り切れ",
    "content_rating": "一般",
    "is_translated": false
  }
}
```

### 2. 搜索结果 `GET /api/v1/search?keyword=xxx`

搜索结果中每个商品的 `status` 和 `condition` 同样会根据语言翻译：

```
GET /api/v1/search?keyword=gundam
Accept-Language: en
```
```json
{
  "code": 0,
  "data": {
    "cached_results": [
      {
        "id": "surugaya_GU375453",
        "title": "...",
        "status": "Available",
        "is_translated": false
      }
    ]
  }
}
```

### 3. 错误信息

所有错误信息自动翻译：

| code | zh-TW | ja | en |
|------|-------|----|----|
| 40001 | 關鍵字被內容政策封鎖 | キーワードはコンテンツポリシーにより制限されています | keyword is blocked by content policy |
| 40002 | 缺少必要參數 | 必須パラメータが不足しています | missing required parameter |
| 40401 | 找不到商品 | 商品が見つかりません | product not found |
| 50001 | 內部伺服器錯誤 | 内部サーバーエラー | internal server error |
| 50003 | 服務暫時無法使用 | サービスが利用できません | service unavailable |

**示例：**
```
GET /api/v1/products/not_exist
Accept-Language: ja
```
```json
{
  "code": 40401,
  "message": "商品が見つかりません",
  "request_id": "7fe26518362a605a"
}
```

---

## 注意事项

1. **标题翻译**：商品被索引后会自动翻译标题和描述，新搜索的商品可能暂时没有翻译版本（`is_translated: false`），此时 `title` 返回日文原文
2. **原文保留**：`title_original` 和 `description_original` 始终是日文原文，可用于展示原始信息
3. **CORS**：`Accept-Language` 已加入 CORS 允许头部列表，跨域请求无需额外配置
