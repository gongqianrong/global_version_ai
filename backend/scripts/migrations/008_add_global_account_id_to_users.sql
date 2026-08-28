-- 添加 global_account_id 字段到 users 表
-- 用于存储国际版用户的真实唯一ID

ALTER TABLE users ADD COLUMN IF NOT EXISTS global_account_id VARCHAR(255);

-- 为现有用户生成 global_account_id（基于email的哈希）
UPDATE users 
SET global_account_id = 'INTL_' || (
    -- 简单哈希：将email转为数字
    CAST(
        (LENGTH(email) * 31 + ASCII(SUBSTRING(email, 1, 1)) * 37 + ASCII(SUBSTRING(email, LENGTH(email), 1)) * 41) 
        % 1000000000 
        AS VARCHAR
    )
)
WHERE global_account_id IS NULL OR global_account_id = '';

-- 创建索引以提升查询性能
CREATE INDEX IF NOT EXISTS idx_users_global_account_id ON users(global_account_id);

-- 添加注释
COMMENT ON COLUMN users.global_account_id IS '国际版用户唯一ID，用于与国内管理端同步';
