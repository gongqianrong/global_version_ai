# 收藏接口 404 问题排查

## 结论

**后端接口正常，404 来自 Vercel 代理层。**

直接调用后端 `http://52.195.4.10:8080/api/v1/favorites/surugaya_606037907` 返回正常（code: 0），
但通过 `https://rakutao-web.vercel.app/api/proxy/favorites/surugaya_606037907` 返回 404 + `{}`。

说明 Vercel 的 `/api/proxy/` 路由没有正确转发带动态路径参数的请求（如 `/favorites/{productID}`）。

## 需要前端修复

Vercel API routes 基于文件系统，需要一个 catch-all 路由来转发所有 `/api/proxy/*` 请求到后端。

例如创建 `pages/api/proxy/[...path].ts`（或 `app/api/proxy/[...path]/route.ts`），将请求转发到 `http://52.195.4.10:8080/api/v1/*`。

---

## 后端收藏接口文档

**Base URL:** `http://52.195.4.10:8080/api/v1`

**所有接口需要 Header:** `Authorization: Bearer <token>`

### 1. 收藏商品

```
POST /favorites/{productID}
```

响应：
```json
{
    "code": 0,
    "data": {
        "id": 5,
        "user_id": 1,
        "product_id": "surugaya_606037907",
        "added_at": "2026-03-03T08:36:56.871509Z"
    }
}
```

> 注意：会验证商品是否存在于 ES 中，不存在会返回 `{"code": 40401, "message": "找不到商品"}`。

### 2. 取消收藏

```
DELETE /favorites/{productID}
```

响应：
```json
{
    "code": 0,
    "data": {
        "product_id": "surugaya_606037907",
        "status": "removed"
    }
}
```

### 3. 收藏列表（分页）

```
GET /favorites?page=1&page_size=20
```

响应：
```json
{
    "code": 0,
    "data": {
        "items": [
            {
                "product_id": "surugaya_606037907",
                "title": "プラモデルツールセット Beginner [PT-100]",
                "title_original": "プラモデルツールセット Beginner [PT-100]",
                "image": "https://cdn.suruga-ya.jp/pics_webp/boxart_m/606037907m.jpg.webp",
                "price_jpy": 891,
                "platform": "surugaya",
                "status": "available",
                "is_translated": false,
                "added_at": "2026-03-03T08:36:56Z"
            }
        ],
        "total": 1,
        "page": 1
    }
}
```

### 4. 批量检查是否已收藏

```
GET /favorites/check?product_ids=id1,id2,id3
```

响应：
```json
{
    "code": 0,
    "data": {
        "surugaya_606037907": true,
        "surugaya_ZSARE20059": false
    }
}
```

---

## 后端测试记录

以下测试于 2026-03-03 直接调用后端完成，全部通过：

| 接口 | 方法 | 状态 | 结果 |
|------|------|------|------|
| `/favorites/surugaya_606037907` | POST | code: 0 | 收藏成功 |
| `/favorites?page=1&page_size=20` | GET | code: 0 | 返回收藏列表 |
| `/favorites/check?product_ids=surugaya_606037907,surugaya_ZSARE20059` | GET | code: 0 | 正确返回 true/false |
| `/favorites/surugaya_606037907` | DELETE | code: 0 | 取消收藏成功 |
