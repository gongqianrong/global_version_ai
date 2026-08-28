#!/bin/bash

# 国际版订单同步接口测试脚本（基于V1.2文档）
# 测试环境：http://52.195.4.10:8080

BASE_URL="http://52.195.4.10:8080"
API_PREFIX="/internal/global/order"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 当前时间戳
TIMESTAMP=$(date +%Y%m%d%H%M%S)

echo "========================================="
echo "国际版订单同步接口测试 (V1.2)"
echo "========================================="
echo ""

# Test 1: 订单同步 - 正常场景
echo -e "${YELLOW}Test 1: 订单同步 - 正常场景${NC}"
SYNC_REQUEST='{
  "requestId": "GLOBAL-SYNC-'$TIMESTAMP'-0001",
  "globalOrderNumber": "G'$TIMESTAMP'0001",
  "globalAccountId": "GU100001",
  "accountAddressId": "",
  "orderAddtime": "2026-08-28 11:20:00",
  "payEffectiveTime": "2026-08-28 12:20:00",
  "orderTotalJp": 10000,
  "orderTotalCn": 500,
  "commissionFeeJp": 1000,
  "commissionFeeCn": 50,
  "handlingFeeJp": 0,
  "handlingFeeCn": 0,
  "orderInpriceJp": 11000,
  "orderInpriceCn": 550,
  "orderRate": 0.05,
  "totalShippingFee": 0,
  "totalShippingFeeCn": 0,
  "orderType": 1,
  "orderPurchaseType": 1,
  "orderMode": 1,
  "orderRemark": "Test order from V1.2 validation",
  "operator": "SYSTEM_GLOBAL",
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD'$TIMESTAMP'000101",
      "platform": 1,
      "goodsMid": "m'$TIMESTAMP'",
      "goodsImg": "https://example.com/item.jpg",
      "goodsName": "测试商品 Test Item",
      "goodsNum": 1,
      "goodsAmountJp": 10000,
      "goodsAmountCn": 500,
      "commissionFeeJp": 1000,
      "commissionFeeCn": 50,
      "handlingFeeJp": 0,
      "handlingFeeCn": 0,
      "goodsUrl": "https://example.com/item/123",
      "sellerId": "seller001",
      "shippingFeeJp": 0,
      "shippingFeeCn": 0,
      "orderPurchaseType": 1,
      "purchaseDirect": 0,
      "discountType": 0
    }
  ]
}'

SYNC_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/sync" \
  -H "Content-Type: application/json" \
  -d "$SYNC_REQUEST")

echo "Response:"
echo "$SYNC_RESPONSE" | jq '.'
echo ""

# 提取订单号用于后续测试
ORDER_NUMBER=$(echo "$SYNC_RESPONSE" | jq -r '.data.orderNumber')
GLOBAL_ORDER_NUMBER=$(echo "$SYNC_RESPONSE" | jq -r '.data.globalOrderNumber')

if [ "$ORDER_NUMBER" != "null" ] && [ "$ORDER_NUMBER" != "" ]; then
  echo -e "${GREEN}✓ 订单同步成功，订单号: $ORDER_NUMBER${NC}"
else
  echo -e "${RED}✗ 订单同步失败${NC}"
fi
echo ""

# Test 2: 幂等性测试 - 重复请求ID
echo -e "${YELLOW}Test 2: 幂等性测试 - 重复请求ID${NC}"
SYNC_IDEMPOTENT_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/sync" \
  -H "Content-Type: application/json" \
  -d "$SYNC_REQUEST")

IDEMPOTENT=$(echo "$SYNC_IDEMPOTENT_RESPONSE" | jq -r '.data.idempotent')
echo "Response:"
echo "$SYNC_IDEMPOTENT_RESPONSE" | jq '.'
echo ""

if [ "$IDEMPOTENT" == "true" ]; then
  echo -e "${GREEN}✓ 幂等性验证成功${NC}"
else
  echo -e "${RED}✗ 幂等性验证失败${NC}"
fi
echo ""

# Test 3: 参数校验 - 缺少globalAccountId
echo -e "${YELLOW}Test 3: 参数校验 - 缺少globalAccountId${NC}"
INVALID_REQUEST='{
  "requestId": "GLOBAL-SYNC-'$TIMESTAMP'-0002",
  "globalOrderNumber": "G'$TIMESTAMP'0002",
  "globalAccountId": "",
  "payEffectiveTime": "2026-08-28 12:20:00",
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD'$TIMESTAMP'000201",
      "platform": 1,
      "goodsMid": "m123",
      "goodsName": "Test",
      "goodsUrl": "https://example.com",
      "discountType": 0
    }
  ]
}'

INVALID_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/sync" \
  -H "Content-Type: application/json" \
  -d "$INVALID_REQUEST")

echo "Response:"
echo "$INVALID_RESPONSE" | jq '.'
ERROR_MSG=$(echo "$INVALID_RESPONSE" | jq -r '.data.message // .message')
echo ""

if [[ "$ERROR_MSG" =~ "globalAccountId" ]]; then
  echo -e "${GREEN}✓ globalAccountId校验成功${NC}"
else
  echo -e "${RED}✗ globalAccountId校验失败${NC}"
fi
echo ""

# Test 4: 参数校验 - goodsName为空
echo -e "${YELLOW}Test 4: 参数校验 - goodsName为空${NC}"
INVALID_GOODSNAME_REQUEST='{
  "requestId": "GLOBAL-SYNC-'$TIMESTAMP'-0003",
  "globalOrderNumber": "G'$TIMESTAMP'0003",
  "globalAccountId": "GU100001",
  "payEffectiveTime": "2026-08-28 12:20:00",
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD'$TIMESTAMP'000301",
      "platform": 1,
      "goodsMid": "m123",
      "goodsName": "",
      "goodsUrl": "https://example.com",
      "discountType": 0
    }
  ]
}'

INVALID_GOODSNAME_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/sync" \
  -H "Content-Type: application/json" \
  -d "$INVALID_GOODSNAME_REQUEST")

echo "Response:"
echo "$INVALID_GOODSNAME_RESPONSE" | jq '.'
ERROR_MSG=$(echo "$INVALID_GOODSNAME_RESPONSE" | jq -r '.data.message // .message')
echo ""

if [[ "$ERROR_MSG" =~ "goodsName" ]]; then
  echo -e "${GREEN}✓ goodsName校验成功${NC}"
else
  echo -e "${RED}✗ goodsName校验失败${NC}"
fi
echo ""

# Test 5: 支付成功同步 - 正常场景
echo -e "${YELLOW}Test 5: 支付成功同步 - 正常场景${NC}"
PAYMENT_REQUEST='{
  "requestId": "GLOBAL-PAY-'$TIMESTAMP'-0001",
  "globalOrderNumber": "'$GLOBAL_ORDER_NUMBER'",
  "paymentNumber": "PAY-TEST-'$TIMESTAMP'-0001",
  "payChannel": "STRIPE",
  "globalOrderPayType": 100,
  "payCurrency": "JPY",
  "payAmount": 11000,
  "payTime": "2026-08-28 11:35:00",
  "operator": "SYSTEM_GLOBAL"
}'

PAYMENT_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/payment-success" \
  -H "Content-Type: application/json" \
  -d "$PAYMENT_REQUEST")

echo "Response:"
echo "$PAYMENT_RESPONSE" | jq '.'
echo ""

SUCCESS=$(echo "$PAYMENT_RESPONSE" | jq -r '.data.success')
if [ "$SUCCESS" == "true" ]; then
  echo -e "${GREEN}✓ 支付同步成功${NC}"
else
  echo -e "${RED}✗ 支付同步失败: $(echo "$PAYMENT_RESPONSE" | jq -r '.data.message')${NC}"
fi
echo ""

# Test 6: 支付幂等性测试
echo -e "${YELLOW}Test 6: 支付幂等性测试 - 重复支付请求${NC}"
PAYMENT_IDEMPOTENT_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/payment-success" \
  -H "Content-Type: application/json" \
  -d "$PAYMENT_REQUEST")

PAYMENT_IDEMPOTENT=$(echo "$PAYMENT_IDEMPOTENT_RESPONSE" | jq -r '.data.idempotent')
echo "Response:"
echo "$PAYMENT_IDEMPOTENT_RESPONSE" | jq '.'
echo ""

if [ "$PAYMENT_IDEMPOTENT" == "true" ]; then
  echo -e "${GREEN}✓ 支付幂等性验证成功${NC}"
else
  echo -e "${RED}✗ 支付幂等性验证失败${NC}"
fi
echo ""

# Test 7: 金额不一致测试
echo -e "${YELLOW}Test 7: 金额不一致测试${NC}"
# 创建新订单用于测试金额不一致
NEW_TIMESTAMP=$(date +%Y%m%d%H%M%S)
SYNC_REQUEST_NEW='{
  "requestId": "GLOBAL-SYNC-'$NEW_TIMESTAMP'-0004",
  "globalOrderNumber": "G'$NEW_TIMESTAMP'0004",
  "globalAccountId": "GU100002",
  "payEffectiveTime": "2026-08-28 12:20:00",
  "orderInpriceJp": 5000,
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD'$NEW_TIMESTAMP'000401",
      "platform": 1,
      "goodsMid": "m456",
      "goodsName": "Test Item 2",
      "goodsUrl": "https://example.com/456",
      "goodsAmountJp": 5000,
      "discountType": 0
    }
  ]
}'

NEW_SYNC_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/sync" \
  -H "Content-Type: application/json" \
  -d "$SYNC_REQUEST_NEW")

NEW_GLOBAL_ORDER_NUMBER=$(echo "$NEW_SYNC_RESPONSE" | jq -r '.data.globalOrderNumber')

# 尝试用不匹配的金额支付
WRONG_AMOUNT_PAYMENT='{
  "requestId": "GLOBAL-PAY-'$NEW_TIMESTAMP'-0004",
  "globalOrderNumber": "'$NEW_GLOBAL_ORDER_NUMBER'",
  "paymentNumber": "PAY-TEST-'$NEW_TIMESTAMP'-0004",
  "payChannel": "STRIPE",
  "globalOrderPayType": 100,
  "payCurrency": "JPY",
  "payAmount": 10000,
  "payTime": "2026-08-28 11:35:00"
}'

WRONG_AMOUNT_RESPONSE=$(curl -s -X POST "${BASE_URL}${API_PREFIX}/payment-success" \
  -H "Content-Type: application/json" \
  -d "$WRONG_AMOUNT_PAYMENT")

echo "Response:"
echo "$WRONG_AMOUNT_RESPONSE" | jq '.'
ERROR_MSG=$(echo "$WRONG_AMOUNT_RESPONSE" | jq -r '.data.message')
echo ""

if [[ "$ERROR_MSG" =~ "金额不一致" ]] || [[ "$ERROR_MSG" =~ "mismatch" ]]; then
  echo -e "${GREEN}✓ 金额校验成功${NC}"
else
  echo -e "${RED}✗ 金额校验失败${NC}"
fi
echo ""

echo "========================================="
echo "测试完成"
echo "========================================="
