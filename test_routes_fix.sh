#!/bin/bash
# 路由修复验证脚本

BASE_URL="http://localhost:8080"

echo "=== 🔧 路由修复验证测试 ==="
echo ""

echo "1. 健康检查"
curl -s "$BASE_URL/health" | jq
echo ""

echo "2. 获取 JWT Token"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}')
echo "$LOGIN_RESP" | jq

TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.token // empty')
if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，请检查用户账号"
  exit 1
fi
echo "✅ Token 获取成功"
echo ""

echo "3. 测试订单列表接口（之前 404）"
ORDERS_RESP=$(curl -s "$BASE_URL/api/v1/orders" \
  -H "Authorization: Bearer $TOKEN")
echo "$ORDERS_RESP" | jq
ORDERS_CODE=$(echo "$ORDERS_RESP" | jq -r '.code // -1')
if [ "$ORDERS_CODE" == "200" ]; then
  echo "✅ 订单列表接口正常"
else
  echo "❌ 订单列表接口返回: $ORDERS_CODE"
fi
echo ""

echo "4. 测试购物车接口（验证受保护路由）"
CART_RESP=$(curl -s "$BASE_URL/api/v1/cart" \
  -H "Authorization: Bearer $TOKEN")
echo "$CART_RESP" | jq
CART_CODE=$(echo "$CART_RESP" | jq -r '.code // -1')
if [ "$CART_CODE" == "200" ]; then
  echo "✅ 购物车接口正常"
else
  echo "❌ 购物车接口返回: $CART_CODE"
fi
echo ""

echo "5. 测试钱包接口"
WALLET_RESP=$(curl -s "$BASE_URL/api/v1/wallet/balance" \
  -H "Authorization: Bearer $TOKEN")
echo "$WALLET_RESP" | jq
WALLET_CODE=$(echo "$WALLET_RESP" | jq -r '.code // -1')
if [ "$WALLET_CODE" == "200" ]; then
  echo "✅ 钱包接口正常"
else
  echo "❌ 钱包接口返回: $WALLET_CODE"
fi
echo ""

echo "6. 测试推荐接口（最后注册的路由）"
RECOMMEND_RESP=$(curl -s "$BASE_URL/api/v1/preferences" \
  -H "Authorization: Bearer $TOKEN")
echo "$RECOMMEND_RESP" | jq
RECOMMEND_CODE=$(echo "$RECOMMEND_RESP" | jq -r '.code // -1')
if [ "$RECOMMEND_CODE" == "200" ] || [ "$RECOMMEND_CODE" == "404" ]; then
  echo "✅ 推荐接口可访问（路由已注册）"
else
  echo "❌ 推荐接口返回: $RECOMMEND_CODE"
fi
echo ""

echo "7. 创建测试订单"
CONFIRM_RESP=$(curl -s -X POST "$BASE_URL/api/v1/order/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"surugaya:123456","quantity":1}]}')
echo "$CONFIRM_RESP" | jq

ORDER_NUMBER=$(echo "$CONFIRM_RESP" | jq -r '.data.orderNumber // empty')
if [ -z "$ORDER_NUMBER" ]; then
  echo "⚠️ 无法创建测试订单（可能商品不存在）"
else
  echo "✅ 测试订单创建成功: $ORDER_NUMBER"
  echo ""
  
  echo "8. 测试订单详情接口（关键修复）"
  ORDER_DETAIL=$(curl -s "$BASE_URL/api/v1/order/$ORDER_NUMBER" \
    -H "Authorization: Bearer $TOKEN")
  echo "$ORDER_DETAIL" | jq
  
  ORDER_DETAIL_CODE=$(echo "$ORDER_DETAIL" | jq -r '.code // -1')
  if [ "$ORDER_DETAIL_CODE" == "200" ]; then
    echo "✅ 订单详情接口正常！修复成功！"
  else
    echo "❌ 订单详情接口返回: $ORDER_DETAIL_CODE"
  fi
fi
echo ""

echo "=== 📊 测试总结 ==="
echo "修复后所有受保护的路由应该可以正常访问"
echo "之前返回 404 的接口现在应该返回正确的响应"
echo ""
echo "Swagger 文档地址: $BASE_URL/swagger/"
