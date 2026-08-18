#!/bin/bash
# 数据库检查脚本 - 直接查询数据验证问题

echo "=== 数据库直接查询验证 ==="
echo ""

# 获取最近的订单
echo "1. 最近5个订单："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  id,
  order_number,
  user_id,
  order_state,
  order_inprice_jp,
  created_at
FROM orders 
ORDER BY created_at DESC 
LIMIT 5;
"

echo ""
echo "2. 最近5笔钱包交易："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  id,
  user_id,
  type,
  amount,
  balance_after,
  description,
  related_order,
  created_at
FROM wallet_transactions 
ORDER BY created_at DESC 
LIMIT 5;
"

echo ""
echo "3. 用户钱包余额："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  user_id,
  balance,
  updated_at
FROM wallets 
ORDER BY updated_at DESC 
LIMIT 5;
"

echo ""
echo "4. 检查是否有事务日志（如果有）："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  NOW() as current_time,
  pg_current_xact_id() as current_transaction_id;
"

echo ""
echo "5. 检查最近的连接和事务："
docker-compose exec -T postgres psql -U postgres -d rakutao -c "
SELECT 
  pid,
  usename,
  application_name,
  state,
  query_start,
  state_change,
  LEFT(query, 100) as query_preview
FROM pg_stat_activity 
WHERE datname = 'rakutao'
AND state != 'idle'
LIMIT 10;
"
