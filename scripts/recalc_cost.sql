-- 一次性脚本：根据最新的模型价格重新计算所有 relay_logs 的费用，并重建统计表
-- 使用方法：
--   1. 先停止 octopus 服务（避免内存缓存与数据库不一致）
--   2. 备份数据库：cp data/data.db data/data.db.bak
--   3. 执行脚本：sqlite3 data/data.db < scripts/recalc_cost.sql
--   4. 重新启动 octopus 服务（启动时会从数据库加载缓存）

BEGIN TRANSACTION;

-- ============================================================
-- 1. 更新 relay_logs 中每条记录的 cost
--    cost = (input_tokens * input_price + output_tokens * output_price) * 1e-6
--    对于 llm_infos 中找不到价格的模型，cost 保持不变
-- ============================================================
UPDATE relay_logs
SET cost = (
    relay_logs.input_tokens * llm_infos.input
    + relay_logs.output_tokens * llm_infos.output
) * 1e-6
FROM llm_infos
WHERE LOWER(relay_logs.actual_model_name) = LOWER(llm_infos.name);

-- ============================================================
-- 2. 重建 stats_totals（全量汇总）
-- ============================================================
DELETE FROM stats_totals;

INSERT INTO stats_totals (id, input_token, output_token, input_cost, output_cost, wait_time, request_success, request_failed)
SELECT
    1,
    COALESCE(SUM(input_tokens), 0),
    COALESCE(SUM(output_tokens), 0),
    COALESCE(SUM(input_tokens * p.input * 1e-6), 0),
    COALESCE(SUM(output_tokens * p.output * 1e-6), 0),
    COALESCE(SUM(use_time), 0),
    COALESCE(SUM(CASE WHEN error = '' OR error IS NULL THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN error != '' AND error IS NOT NULL THEN 1 ELSE 0 END), 0)
FROM relay_logs r
LEFT JOIN llm_infos p ON LOWER(r.actual_model_name) = LOWER(p.name);

-- ============================================================
-- 3. 重建 stats_dailies（按日汇总）
-- ============================================================
DELETE FROM stats_dailies;

INSERT INTO stats_dailies (date, input_token, output_token, input_cost, output_cost, wait_time, request_success, request_failed)
SELECT
    STRFTIME('%Y%m%d', r.time, 'unixepoch', 'localtime'),
    COALESCE(SUM(r.input_tokens), 0),
    COALESCE(SUM(r.output_tokens), 0),
    COALESCE(SUM(r.input_tokens * p.input * 1e-6), 0),
    COALESCE(SUM(r.output_tokens * p.output * 1e-6), 0),
    COALESCE(SUM(r.use_time), 0),
    COALESCE(SUM(CASE WHEN r.error = '' OR r.error IS NULL THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN r.error != '' AND r.error IS NOT NULL THEN 1 ELSE 0 END), 0)
FROM relay_logs r
LEFT JOIN llm_infos p ON LOWER(r.actual_model_name) = LOWER(p.name)
GROUP BY STRFTIME('%Y%m%d', r.time, 'unixepoch', 'localtime');

-- ============================================================
-- 4. 重建 stats_hourlies（按小时汇总，只保留今天的数据）
-- ============================================================
DELETE FROM stats_hourlies;

INSERT INTO stats_hourlies (hour, date, input_token, output_token, input_cost, output_cost, wait_time, request_success, request_failed)
SELECT
    CAST(STRFTIME('%H', r.time, 'unixepoch', 'localtime') AS INTEGER),
    STRFTIME('%Y%m%d', r.time, 'unixepoch', 'localtime'),
    COALESCE(SUM(r.input_tokens), 0),
    COALESCE(SUM(r.output_tokens), 0),
    COALESCE(SUM(r.input_tokens * p.input * 1e-6), 0),
    COALESCE(SUM(r.output_tokens * p.output * 1e-6), 0),
    COALESCE(SUM(r.use_time), 0),
    COALESCE(SUM(CASE WHEN r.error = '' OR r.error IS NULL THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN r.error != '' AND r.error IS NOT NULL THEN 1 ELSE 0 END), 0)
FROM relay_logs r
LEFT JOIN llm_infos p ON LOWER(r.actual_model_name) = LOWER(p.name)
WHERE STRFTIME('%Y%m%d', r.time, 'unixepoch', 'localtime') = STRFTIME('%Y%m%d', 'now', 'localtime')
GROUP BY CAST(STRFTIME('%H', r.time, 'unixepoch', 'localtime') AS INTEGER);

-- ============================================================
-- 5. 重建 stats_channels（按渠道汇总）
-- ============================================================
DELETE FROM stats_channels;

INSERT INTO stats_channels (channel_id, input_token, output_token, input_cost, output_cost, wait_time, request_success, request_failed)
SELECT
    r.channel_id,
    COALESCE(SUM(r.input_tokens), 0),
    COALESCE(SUM(r.output_tokens), 0),
    COALESCE(SUM(r.input_tokens * p.input * 1e-6), 0),
    COALESCE(SUM(r.output_tokens * p.output * 1e-6), 0),
    COALESCE(SUM(r.use_time), 0),
    COALESCE(SUM(CASE WHEN r.error = '' OR r.error IS NULL THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN r.error != '' AND r.error IS NOT NULL THEN 1 ELSE 0 END), 0)
FROM relay_logs r
LEFT JOIN llm_infos p ON LOWER(r.actual_model_name) = LOWER(p.name)
GROUP BY r.channel_id;

-- ============================================================
-- 6. 重建 stats_api_keys（按 API Key 汇总）
--    relay_logs 中没有 api_key_id 字段，需要通过 api_key_name 关联
-- ============================================================
DELETE FROM stats_api_keys;

INSERT INTO stats_api_keys (api_key_id, input_token, output_token, input_cost, output_cost, wait_time, request_success, request_failed)
SELECT
    ak.id,
    COALESCE(SUM(r.input_tokens), 0),
    COALESCE(SUM(r.output_tokens), 0),
    COALESCE(SUM(r.input_tokens * p.input * 1e-6), 0),
    COALESCE(SUM(r.output_tokens * p.output * 1e-6), 0),
    COALESCE(SUM(r.use_time), 0),
    COALESCE(SUM(CASE WHEN r.error = '' OR r.error IS NULL THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN r.error != '' AND r.error IS NOT NULL THEN 1 ELSE 0 END), 0)
FROM relay_logs r
JOIN api_keys ak ON r.request_api_key_name = ak.name
LEFT JOIN llm_infos p ON LOWER(r.actual_model_name) = LOWER(p.name)
GROUP BY ak.id;

COMMIT;

-- 验证结果
SELECT '=== relay_logs cost 样本 ===' AS info;
SELECT actual_model_name, input_tokens, output_tokens, cost FROM relay_logs LIMIT 10;

SELECT '=== stats_totals ===' AS info;
SELECT * FROM stats_totals;

SELECT '=== stats_dailies ===' AS info;
SELECT * FROM stats_dailies ORDER BY date DESC LIMIT 10;

SELECT '=== stats_channels ===' AS info;
SELECT * FROM stats_channels;
