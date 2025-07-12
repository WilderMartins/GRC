-- Migração Inicial: Reversão da Criação das Tabelas Base

-- A ordem de drop pode importar por causa de Foreign Keys.
-- Usar CASCADE para simplificar, mas em produção, a ordem explícita é mais segura
-- se CASCADE não for o comportamento desejado para todas as FKs.

DROP TABLE IF EXISTS approval_workflows CASCADE;
DROP TABLE IF EXISTS risk_stakeholders CASCADE;
DROP TABLE IF EXISTS webhook_configurations CASCADE;
DROP TABLE IF EXISTS audit_assessments CASCADE;
DROP TABLE IF EXISTS audit_controls CASCADE;
DROP TABLE IF EXISTS audit_frameworks CASCADE;
DROP TABLE IF EXISTS vulnerabilities CASCADE;
DROP TABLE IF EXISTS risks CASCADE;
DROP TABLE IF EXISTS identity_providers CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;

-- (Fim do arquivo 000001_create_initial_tables.down.sql)
