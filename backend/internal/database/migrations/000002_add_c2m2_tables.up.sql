-- Migração C2M2: Criação das tabelas para domínios, práticas e avaliações C2M2

-- Domínios C2M2
CREATE TABLE IF NOT EXISTS c2m2_domains (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Práticas C2M2
CREATE TABLE IF NOT EXISTS c2m2_practices (
    id UUID PRIMARY KEY,
    domain_id UUID NOT NULL REFERENCES c2m2_domains(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    target_mil INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_c2m2_practices_domain_id ON c2m2_practices(domain_id);

-- Avaliações de Práticas C2M2
CREATE TABLE IF NOT EXISTS c2m2_practice_evaluations (
    id UUID PRIMARY KEY,
    audit_assessment_id UUID NOT NULL REFERENCES audit_assessments(id) ON DELETE CASCADE,
    practice_id UUID NOT NULL REFERENCES c2m2_practices(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- not_implemented, partially_implemented, fully_implemented
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    UNIQUE (audit_assessment_id, practice_id) -- Garantir uma avaliação por prática por assessment
);
CREATE INDEX IF NOT EXISTS idx_c2m2_practice_evaluations_assessment_id ON c2m2_practice_evaluations(audit_assessment_id);
CREATE INDEX IF NOT EXISTS idx_c2m2_practice_evaluations_practice_id ON c2m2_practice_evaluations(practice_id);

-- Adicionar a relação `has-many` de `AuditAssessment` para `C2M2PracticeEvaluation`
-- A constraint de chave estrangeira já foi adicionada na criação da tabela c2m2_practice_evaluations.
-- A adição da coluna em `audit_assessments` já foi feita na migração 000001.
-- Esta migração é focada nas novas tabelas.
