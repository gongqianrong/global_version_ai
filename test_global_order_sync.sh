#!/bin/bash
# 国际版订单同步接口测试脚本

BASE_URL="${BASE_URL:-http://localhost:8080}"
TIMESTAMP=$(date +%s)
TEST_USER_ID="100001"

echo "=== 🧪 国际版订单同步接口测试 ==="
echo "BASE_URL: $BASE_URL"
echo "测试时间: $(date)"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
PASSED=0
FAILED=0

# JSON 提取函数（不依赖 jq）
extract_json_field() {
    local json="$1"
    local field="$2"
    local default="${3:-}"
    
    # 简单的 JSON 提取，使用 grep + sed
    local value=$(echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*[^,}]*" | sed "s/\"$field\"[[:space:]]*:[[:space:]]*//; s/[\",]//g")
    if [ -z "$value" ]; then
        echo "$default"
    else
        echo "$value"
    fi
}

# 测试函数
test_case() {
    local name="$1"
    local expected_success="$2"
    local response="$3"
    
    local code=$(extract_json_field "$response" "code" "-1")
    local data_success=$(extract_json_field "$response" "success" "false")
    
    if [ "$code" == "200" ] && [ "$data_success" == "$expected_success" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $name"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}❌ FAIL${NC}: $name"
        echo "   Expected success: $expected_success"
        echo "   Actual code: $code, data.success: $data_success"
        echo "   Response: $response"
        ((FAILED++))
        return 1
    fi
}

# 获取 ISO 时间（跨平台）
get_iso_time() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        date -u +"%Y-%m-%dT%H:%M:%S+00:00"
    else
        # Linux
        date -u +"%Y-%m-%dT%H:%M:%S+00:00"
    fi
}

get_iso_time_offset() {
    local hours="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        date -u -v+${hours}H +"%Y-%m-%dT%H:%M:%S+00:00" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%S+00:00"
    else
        # Linux
        date -u -d "+${hours} hour" +"%Y-%m-%dT%H:%M:%S+00:00"
    fi
}

echo "=== 测试 1: 订单同步 - 首次同步（应该成功）==="
GLOBAL_ORDER_1="G${TIMESTAMP}0001"
REQUEST_ID_1="REQ-SYNC-${TIMESTAMP}-0001"

SYNC_RESP_1=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/sync" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"$REQUEST_ID_1\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER_1\",
    \"globalAccountId\": \"GU$TEST_USER_ID\",
    \"accountInfoId\": \"$TEST_USER_ID\",
    \"accountAddressId\": \"ADDR$TEST_USER_ID\",
    \"orderAddtime\": \"$(get_iso_time)\",
    \"payEffectiveTime\": \"$(get_iso_time_offset 1)\",
    \"orderTotalJp\": 10000,
    \"orderTotalCn\": 500,
    \"commissionFeeJp\": 1000,
    \"commissionFeeCn\": 50,
    \"handlingFeeJp\": 0,
    \"handlingFeeCn\": 0,
    \"orderInpriceJp\": 11000,
    \"orderInpriceCn\": 550,
    \"orderRate\": 0.05,
    \"totalShippingFee\": 0,
    \"totalShippingFeeCn\": 0,
    \"orderType\": 1,
    \"orderPurchaseType\": 1,
    \"orderMode\": 1,
    \"orderRemark\": \"测试订单 - 首次同步\",
    \"operator\": \"SYSTEM_TEST\",
    \"globalOrderPayType\": 100,
    \"detailList\": [
      {
        \"globalOrderDetailNumber\": \"${GLOBAL_ORDER_1}D01\",
        \"platform\": 1,
        \"goodsMid\": \"test_item_$TIMESTAMP\",
        \"goodsImg\": \"https://example.com/test.jpg\",
        \"goodsName\": \"测试商品\",
        \"goodsNum\": 1,
        \"goodsAmountJp\": 10000,
        \"goodsAmountCn\": 500,
        \"commissionFeeJp\": 1000,
        \"commissionFeeCn\": 50,
        \"handlingFeeJp\": 0,
        \"handlingFeeCn\": 0,
        \"goodsUrl\": \"https://example.com/test\",
        \"sellerId\": \"test_seller\",
        \"shippingFeeJp\": 0,
        \"shippingFeeCn\": 0,
        \"orderPurchaseType\": 1,
        \"purchaseDirect\": 0,
        \"discountType\": 0
      }
    ]
  }")

echo "$SYNC_RESP_1"
test_case "首次订单同步" "true" "$SYNC_RESP_1"

# 提取订单号
LOCAL_ORDER_NUMBER=$(extract_json_field "$SYNC_RESP_1" "orderNumber")
if [ -z "$LOCAL_ORDER_NUMBER" ] || [ "$LOCAL_ORDER_NUMBER" == "null" ]; then
    echo -e "${RED}❌ 无法获取本地订单号，后续测试中止${NC}"
    echo "Response: $SYNC_RESP_1"
    exit 1
fi
echo "本地订单号: $LOCAL_ORDER_NUMBER"
echo ""

echo "=== 测试 2: 订单同步 - 幂等性测试（相同 requestId，应该返回幂等）==="
SYNC_RESP_2=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/sync" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"$REQUEST_ID_1\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER_1\",
    \"globalAccountId\": \"GU$TEST_USER_ID\",
    \"accountInfoId\": \"$TEST_USER_ID\",
    \"accountAddressId\": \"ADDR$TEST_USER_ID\",
    \"orderAddtime\": \"$(get_iso_time)\",
    \"payEffectiveTime\": \"$(get_iso_time_offset 1)\",
    \"orderTotalJp\": 10000,
    \"orderTotalCn\": 500,
    \"commissionFeeJp\": 1000,
    \"commissionFeeCn\": 50,
    \"handlingFeeJp\": 0,
    \"handlingFeeCn\": 0,
    \"orderInpriceJp\": 11000,
    \"orderInpriceCn\": 550,
    \"orderRate\": 0.05,
    \"totalShippingFee\": 0,
    \"totalShippingFeeCn\": 0,
    \"orderType\": 1,
    \"orderPurchaseType\": 1,
    \"orderMode\": 1,
    \"orderRemark\": \"测试订单 - 幂等性\",
    \"operator\": \"SYSTEM_TEST\",
    \"globalOrderPayType\": 100,
    \"detailList\": [
      {
        \"globalOrderDetailNumber\": \"${GLOBAL_ORDER_1}D01\",
        \"platform\": 1,
        \"goodsMid\": \"test_item_$TIMESTAMP\",
        \"goodsImg\": \"https://example.com/test.jpg\",
        \"goodsName\": \"测试商品\",
        \"goodsNum\": 1,
        \"goodsAmountJp\": 10000,
        \"goodsAmountCn\": 500,
        \"commissionFeeJp\": 1000,
        \"commissionFeeCn\": 50,
        \"handlingFeeJp\": 0,
        \"handlingFeeCn\": 0,
        \"goodsUrl\": \"https://example.com/test\",
        \"sellerId\": \"test_seller\",
        \"shippingFeeJp\": 0,
        \"shippingFeeCn\": 0,
        \"orderPurchaseType\": 1,
        \"purchaseDirect\": 0,
        \"discountType\": 0
      }
    ]
  }")

echo "$SYNC_RESP_2"
IDEMPOTENT=$(extract_json_field "$SYNC_RESP_2" "idempotent" "false")
if [ "$IDEMPOTENT" == "true" ]; then
    echo -e "${GREEN}✅ PASS${NC}: 幂等性测试（正确返回 idempotent=true）"
    ((PASSED++))
else
    echo -e "${RED}❌ FAIL${NC}: 幂等性测试（期望 idempotent=true，实际 $IDEMPOTENT）"
    ((FAILED++))
fi
echo ""

echo "=== 测试 3: 支付同步 - 首次支付（应该成功）==="
PAY_REQUEST_ID="REQ-PAY-${TIMESTAMP}-0001"
PAYMENT_NUMBER="PAY-TEST-${TIMESTAMP}"

PAY_RESP_1=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/payment-success" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"$PAY_REQUEST_ID\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER_1\",
    \"paymentNumber\": \"$PAYMENT_NUMBER\",
    \"payChannel\": \"TEST\",
    \"payWay\": 1,
    \"payAmountJp\": 11000,
    \"payAmountCn\": 550,
    \"paySeccussTime\": \"$(get_iso_time)\"
  }")

echo "$PAY_RESP_1"
test_case "首次支付同步" "true" "$PAY_RESP_1"
echo ""

echo "=== 测试 4: 支付同步 - 幂等性测试（相同 requestId）==="
PAY_RESP_2=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/payment-success" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"$PAY_REQUEST_ID\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER_1\",
    \"paymentNumber\": \"$PAYMENT_NUMBER\",
    \"payChannel\": \"TEST\",
    \"payWay\": 1,
    \"payAmountJp\": 11000,
    \"payAmountCn\": 550,
    \"paySeccussTime\": \"$(get_iso_time)\"
  }")

echo "$PAY_RESP_2"
PAY_IDEMPOTENT=$(extract_json_field "$PAY_RESP_2" "idempotent" "false")
if [ "$PAY_IDEMPOTENT" == "true" ]; then
    echo -e "${GREEN}✅ PASS${NC}: 支付幂等性测试（正确返回 idempotent=true）"
    ((PASSED++))
else
    echo -e "${RED}❌ FAIL${NC}: 支付幂等性测试（期望 idempotent=true，实际 $PAY_IDEMPOTENT）"
    ((FAILED++))
fi
echo ""

echo "=== 测试 5: 金额不匹配检测（应该失败）==="
PAY_RESP_WRONG=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/payment-success" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"REQ-PAY-${TIMESTAMP}-WRONG\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER_1\",
    \"paymentNumber\": \"PAY-WRONG-${TIMESTAMP}\",
    \"payChannel\": \"TEST\",
    \"payWay\": 1,
    \"payAmountJp\": 9999,
    \"payAmountCn\": 500,
    \"paySeccussTime\": \"$(get_iso_time)\"
  }")

echo "$PAY_RESP_WRONG"
WRONG_SUCCESS=$(extract_json_field "$PAY_RESP_WRONG" "success" "true")
if [ "$WRONG_SUCCESS" == "false" ]; then
    echo -e "${GREEN}✅ PASS${NC}: 金额不匹配检测（正确拒绝）"
    ((PASSED++))
else
    echo -e "${RED}❌ FAIL${NC}: 金额不匹配检测（应该拒绝但返回成功）"
    ((FAILED++))
fi
echo ""

echo "=== 测试 6: 不存在的订单支付（应该失败）==="
PAY_RESP_NOTFOUND=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/payment-success" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"REQ-PAY-${TIMESTAMP}-NOTFOUND\",
    \"globalOrderNumber\": \"G99999999999\",
    \"paymentNumber\": \"PAY-NOTFOUND-${TIMESTAMP}\",
    \"payChannel\": \"TEST\",
    \"payWay\": 1,
    \"payAmountJp\": 11000,
    \"payAmountCn\": 550,
    \"paySeccussTime\": \"$(get_iso_time)\"
  }")

echo "$PAY_RESP_NOTFOUND"
NOTFOUND_SUCCESS=$(extract_json_field "$PAY_RESP_NOTFOUND" "success" "true")
if [ "$NOTFOUND_SUCCESS" == "false" ]; then
    echo -e "${GREEN}✅ PASS${NC}: 不存在订单检测（正确拒绝）"
    ((PASSED++))
else
    echo -e "${RED}❌ FAIL${NC}: 不存在订单检测（应该拒绝但返回成功）"
    ((FAILED++))
fi
echo ""

echo "=== 测试 7: 缺少必填字段（应该返回错误）==="
INVALID_RESP=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/sync" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"REQ-INVALID-${TIMESTAMP}\",
    \"globalOrderNumber\": \"\"
  }")

echo "$INVALID_RESP"
INVALID_CODE=$(extract_json_field "$INVALID_RESP" "code" "200")
if [ "$INVALID_CODE" != "200" ]; then
    echo -e "${GREEN}✅ PASS${NC}: 缺少必填字段检测（正确返回错误）"
    ((PASSED++))
else
    echo -e "${RED}❌ FAIL${NC}: 缺少必填字段检测（应该返回错误但返回成功）"
    ((FAILED++))
fi
echo ""

echo "========================================="
echo "=== 📊 测试结果汇总 ==="
echo "========================================="
echo -e "通过: ${GREEN}$PASSED${NC}"
echo -e "失败: ${RED}$FAILED${NC}"
echo "总计: $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}❌ 有 $FAILED 个测试失败${NC}"
    exit 1
fi
