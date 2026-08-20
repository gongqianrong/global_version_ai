#!/bin/bash
# 数据库验证脚本 - 检查国际版订单同步的数据一致性

echo "=== 🗄️ 数据库验证 - 国际版订单同步 ==="
echo ""

echo "1. 检查 global_order_records 表"
echo "最近5条国际版订单记录："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  id,
  global_order_number,
  order_number,
  order_state,
  pay_sync_state,
  created_at
FROM global_order_records 
ORDER BY created_at DESC 
LIMIT 5;
"

echo ""
echo "2. 检查对应的本地订单"
echo "最近5个通过国际版同步的订单："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  o.id,
  o.order_number,
  o.order_state,
  o.order_inprice_jp,
  o.order_purchase_type,
  gor.global_order_number,
  gor.pay_sync_state,
  o.created_at
FROM orders o
JOIN global_order_records gor ON o.order_number = gor.order_number
ORDER BY o.created_at DESC 
LIMIT 5;
"

echo ""
echo "3. 检查支付记录"
echo "最近5条国际版支付记录："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  id,
  global_order_number,
  payment_number,
  pay_channel,
  pay_amount_jp,
  pay_amount_cn,
  created_at
FROM global_order_payments 
ORDER BY created_at DESC 
LIMIT 5;
"

echo ""
echo "4. 数据一致性检查"
echo "检查是否有订单状态不匹配的情况："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  gor.global_order_number,
  gor.order_number,
  gor.order_state as recorded_state,
  o.order_state as actual_state,
  gor.pay_sync_state,
  CASE 
    WHEN gor.order_state = o.order_state THEN '✅ 一致'
    ELSE '❌ 不一致'
  END as consistency_check
FROM global_order_records gor
JOIN orders o ON gor.order_number = o.order_number
ORDER BY gor.created_at DESC
LIMIT 10;
"

echo ""
echo "5. 幂等性检查"
echo "检查是否有重复的 globalOrderNumber："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  global_order_number,
  COUNT(*) as count,
  CASE 
    WHEN COUNT(*) = 1 THEN '✅ 正常'
    ELSE '❌ 重复'
  END as status
FROM global_order_records
GROUP BY global_order_number
HAVING COUNT(*) > 1;
"

echo ""
echo "6. 支付状态统计"
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  pay_sync_state,
  COUNT(*) as count,
  CASE 
    WHEN pay_sync_state = 0 THEN '未支付'
    WHEN pay_sync_state = 1 THEN '已支付'
    WHEN pay_sync_state = 2 THEN '异常'
    ELSE '未知'
  END as status_desc
FROM global_order_records
GROUP BY pay_sync_state;
"

echo ""
echo "7. 订单详情数量检查"
echo "检查订单和明细的数量是否匹配："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  o.order_number,
  gor.global_order_number,
  COUNT(od.id) as detail_count,
  CASE 
    WHEN COUNT(od.id) > 0 THEN '✅ 有明细'
    ELSE '❌ 无明细'
  END as has_details
FROM orders o
JOIN global_order_records gor ON o.order_number = gor.order_number
LEFT JOIN order_details od ON o.id = od.order_id
GROUP BY o.order_number, gor.global_order_number
ORDER BY o.created_at DESC
LIMIT 5;
"

echo ""
echo "=== ✅ 数据库验证完成 ==="
