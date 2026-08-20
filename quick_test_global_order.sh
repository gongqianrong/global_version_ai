#!/bin/bash
# 快速测试脚本 - 国际版订单同步接口

echo "=== 🧪 快速测试国际版订单同步接口 ==="

# 检测服务器 IP
if command -v curl &> /dev/null; then
    SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
    if [ -z "$SERVER_IP" ]; then
        SERVER_IP="localhost"
    fi
else
    SERVER_IP="localhost"
fi

BASE_URL="http://${SERVER_IP}:8080"

echo "服务器地址: $BASE_URL"
echo ""

# 测试 1：健康检查
echo "1. 健康检查..."
HEALTH=$(curl -s "$BASE_URL/health" 2>&1)
if [ $? -eq 0 ]; then
    echo "✅ 服务正常运行"
    echo "   Response: $HEALTH"
else
    echo "❌ 服务无法访问"
    echo "   Error: $HEALTH"
    exit 1
fi
echo ""

# 测试 2：订单同步
echo "2. 测试订单同步..."
TIMESTAMP=$(date +%s)
GLOBAL_ORDER="GTEST${TIMESTAMP}"

SYNC_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/sync" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"TEST-SYNC-${TIMESTAMP}\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER\",
    \"globalAccountId\": \"GU100001\",
    \"accountInfoId\": \"100001\",
    \"accountAddressId\": \"ADDR001\",
    \"orderAddtime\": \"$(date -u +%Y-%m-%dT%H:%M:%S+00:00)\",
    \"payEffectiveTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S+00:00)\",
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
    \"orderRemark\": \"快速测试订单\",
    \"operator\": \"QUICK_TEST\",
    \"globalOrderPayType\": 100,
    \"detailList\": [{
        \"globalOrderDetailNumber\": \"${GLOBAL_ORDER}D01\",
        \"platform\": 1,
        \"goodsMid\": \"test_${TIMESTAMP}\",
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
    }]
  }")

echo "$SYNC_RESULT"

# 检查成功标识
if echo "$SYNC_RESULT" | grep -q '"success"[[:space:]]*:[[:space:]]*true'; then
    echo "✅ 订单同步成功"
else
    echo "❌ 订单同步失败"
fi
echo ""

# 测试 3：支付同步
echo "3. 测试支付同步..."
PAY_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/internal/global/order/payment-success" \
  -H "Content-Type: application/json" \
  -d "{
    \"requestId\": \"TEST-PAY-${TIMESTAMP}\",
    \"globalOrderNumber\": \"$GLOBAL_ORDER\",
    \"paymentNumber\": \"PAY-TEST-${TIMESTAMP}\",
    \"payChannel\": \"TEST\",
    \"payWay\": 1,
    \"payAmountJp\": 11000,
    \"payAmountCn\": 550,
    \"paySeccussTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S+00:00)\"
  }")

echo "$PAY_RESULT"

if echo "$PAY_RESULT" | grep -q '"success"[[:space:]]*:[[:space:]]*true'; then
    echo "✅ 支付同步成功"
else
    echo "❌ 支付同步失败"
fi
echo ""

echo "=== ✅ 快速测试完成 ==="
echo ""
echo "提示: 运行完整测试请使用: bash test_global_order_sync.sh"
echo "      检查数据库请使用: bash verify_global_order_data.sh"
