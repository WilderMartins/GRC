-- Migração C2M2: Reversão da criação das tabelas C2M2

-- A ordem importa se não usar CASCADE.
DROP TABLE IF EXISTS c2m2_practice_evaluations;
DROP TABLE IF EXISTS c2m2_practices;
DROP TABLE IF EXISTS c2m2_domains;
