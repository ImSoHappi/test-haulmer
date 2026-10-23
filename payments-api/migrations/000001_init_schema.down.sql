-- 000001_init_schema.down.sql
-- Revert initial schema definition

DROP TABLE IF EXISTS transaction_events CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS payment_intents CASCADE;
DROP TABLE IF EXISTS merchants CASCADE;
