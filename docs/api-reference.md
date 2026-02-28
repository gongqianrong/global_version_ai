# 新增接口文档

## 1. 骏河屋 - 一级分类

### `GET /api/v1/surugaya/categories`

获取骏河屋一级商品分类菜单。无参数。

**上游接口:** `GET /suruga/category/search/adv`

**缓存:** 1 小时 + 24 小时兜底

**响应示例:**
```json
{
  "code": 0,
  "data": {
    "category": [
      { "category_id": "5", "category_name": "おもちゃホビー" },
      { "category_id": "2", "category_name": "ゲーム" },
      { "category_id": "7", "category_name": "本" }
    ],
    "platform": 2
  },
  "request_id": "e2f8516f0fdfad04"
}
```

---

## 2. 骏河屋 - 子级分类

### `GET /api/v1/surugaya/categories/{id}`

根据父分类 ID 获取子级分类列表。

**上游接口:** `GET /suruga/category/search/adv/{category_id}`

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| id | Path | 是 | 父分类 ID |

**缓存:** 1 小时 + 24 小时兜底

**响应示例:**
```json
{
  "code": 0,
  "data": {
    "category": [
      { "category_id": "501", "category_name": "フィギュア" },
      { "category_id": "502", "category_name": "プラモデル" }
    ]
  },
  "request_id": "ec2b72c5d158e2eb"
}
```
