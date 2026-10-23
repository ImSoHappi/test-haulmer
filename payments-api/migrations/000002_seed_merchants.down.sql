-- 000002_seed_merchants.down.sql
-- Revert seeded merchants and any dependent test data

DELETE FROM merchants WHERE id IN ('m_1001', 'm_1002');
