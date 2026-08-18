#!/bin/bash
# 订单支付问题调试脚本

BASE_URL="http://localhost:8080"

echo "=== 1. 健康检查 ==="
curl -s "$BASE_URL/health"
echo -e "\n"

echo "=== 2. 用户登录 ==="
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}')
echo "$LOGIN_RESP"

TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.token // empty')
if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，无法获取 token"
  exit 1
fi
echo "✅ Token: ${TOKEN:0:20}..."
echo ""

echo "=== 3. 查看钱包余额（支付前）==="
WALLET_BEFORE=$(curl -s "$BASE_URL/api/v1/wallet" \
  -H "Authorization: Bearer $TOKEN")
echo "$WALLET_BEFORE"
BALANCE_BEFORE=$(echo "$WALLET_BEFORE" | jq -r '.data.balance // 0')
echo "💰 支付前余额: $BALANCE_BEFORE"
echo ""

echo "=== 4. 确认订单（创建订单）==="
CONFIRM_RESP=$(curl -s -X POST "$BASE_URL/api/v1/order/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"product_id": "surugaya:123456", "quantity": 1}
    ],
    "order_paytype": 1,
    "order_remark": "测试订单"
  }')
echo "$CONFIRM_RESP"

ORDER_NUMBER=$(echo "$CONFIRM_RESP" | jq -r '.data.orderNumber // empty')
if [ -z "$ORDER_NUMBER" ]; then
  echo "❌ 订单创建失败"
  exit 1
fi
echo "✅ 订单号: $ORDER_NUMBER"
echo ""

echo "=== 5. 查询订单详情（支付前）==="
ORDER_DETAIL_BEFORE=$(curl -s "$BASE_URL/api/v1/order/$ORDER_NUMBER" \
  -H "Authorization: Bearer $TOKEN")
echo "$ORDER_DETAIL_BEFORE"
ORDER_STATE_BEFORE=$(echo "$ORDER_DETAIL_BEFORE" | jq -r '.data.orderState // -1')
echo "📦 订单状态（支付前）: $ORDER_STATE_BEFORE (应该是 0=Pending)"
echo ""

echo "=== 6. 支付订单 ==="
PAY_RESP=$(curl -s -X POST "$BASE_URL/api/v1/order/pay" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"order_number\":\"$ORDER_NUMBER\"}")
echo "$PAY_RESP"

PAY_SUCCESS=$(echo "$PAY_RESP" | jq -r '.code // -1')
if [ "$PAY_SUCCESS" != "200" ]; then
  echo "❌ 支付失败: code=$PAY_SUCCESS"
  echo "错误信息: $(echo "$PAY_RESP" | jq -r '.message // "unknown"')"
  exit 1
fi
echo "✅ 支付API返回成功"
echo ""

echo "=== 7. 查询订单详情（支付后）==="
sleep 2
ORDER_DETAIL_AFTER=$(curl -s "$BASE_URL/api/v1/order/$ORDER_NUMBER" \
  -H "Authorization: Bearer $TOKEN")
echo "$ORDER_DETAIL_AFTER"

ORDER_STATE_AFTER=$(echo "$ORDER_DETAIL_AFTER" | jq -r '.data.orderState // -1')
echo "📦 订单状态（支付后）: $ORDER_STATE_AFTER (应该是 3=Paid)"

if [ "$ORDER_STATE_AFTER" == "-1" ] || [ "$ORDER_STATE_AFTER" == "null" ]; then
  echo "❌ 订单不存在或查询失败！"
  echo "完整响应: $ORDER_DETAIL_AFTER"
fi
echo ""

echo "=== 8. 查看钱包余额（支付后）==="
WALLET_AFTER=$(curl -s "$BASE_URL/api/v1/wallet" \
  -H "Authorization: Bearer $TOKEN")
echo "$WALLET_AFTER"
BALANCE_AFTER=$(echo "$WALLET_AFTER" | jq -r '.data.balance // 0')
echo "💰 支付后余额: $BALANCE_AFTER"
echo ""

echo "=== 9. 钱包交易记录 ==="
TRANSACTIONS=$(curl -s "$BASE_URL/api/v1/wallet/transactions?limit=5" \
  -H "Authorization: Bearer $TOKEN")
echo "$TRANSACTIONS"
echo ""

echo "=== 10. 结果分析 ==="
echo "支付前余额: $BALANCE_BEFORE"
echo "支付后余额: $BALANCE_AFTER"
echo "订单状态（支付前）: $ORDER_STATE_BEFORE"
echo "订单状态（支付后）: $ORDER_STATE_AFTER"

BALANCE_DIFF=$((BALANCE_BEFORE - BALANCE_AFTER))
echo "余额变化: $BALANCE_DIFF"

if [ "$ORDER_STATE_AFTER" != "3" ]; then
  echo "❌ 问题：订单状态未更新为已支付！"
fi

if [ "$BALANCE_DIFF" == "0" ]; then
  echo "❌ 问题：钱包余额未扣减！"
fi

if [ "$ORDER_STATE_AFTER" == "3" ] && [ "$BALANCE_DIFF" -gt "0" ]; then
  echo "✅ 支付成功！订单状态和钱包余额都已更新"
else
  echo "❌ 支付失败！存在数据不一致问题"
fi
