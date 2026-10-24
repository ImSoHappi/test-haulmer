-- 000002_seed_merchants.up.sql
-- Seed initial merchants for testing and evaluation

INSERT INTO merchants (id, name, status) VALUES
    ('1001', 'Comercio Activo Principal', 'ACTIVE'),
    ('1002', 'Comercio Inactivo Test', 'INACTIVE')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    updated_at = now();
