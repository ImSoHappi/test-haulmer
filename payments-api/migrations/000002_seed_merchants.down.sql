-- 000002_seed_merchants.down.sql
-- Revert seeded merchants and any dependent test data

DELETE FROM merchants WHERE id IN ('1001', '1002');
