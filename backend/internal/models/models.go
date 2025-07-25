package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Custom type for Impact, Probability, Status, etc. to enforce specific values
type RiskImpact string
type RiskProbability string
type RiskStatus string
type VulnerabilitySeverity string
type VulnerabilityStatus string
type ApprovalStatus string
type UserRole string
type AuditControlStatus string
type RiskCategory string

const (
	ImpactLow       RiskImpact = "Baixo"
	ImpactMedium    RiskImpact = "Médio"
	ImpactHigh      RiskImpact = "Alto"
	ImpactCritical  RiskImpact = "Crítico"

	ProbabilityLow      RiskProbability = "Baixo" // Ajustado para masculino e capitalizado
	ProbabilityMedium   RiskProbability = "Médio" // Ajustado para masculino e capitalizado
	ProbabilityHigh     RiskProbability = "Alto"  // Ajustado para masculino e capitalizado
	ProbabilityCritical RiskProbability = "Crítico"// Ajustado para masculino e capitalizado

	// RiskLevel é o nível de risco calculado (ex: Baixo, Moderado, Alto, Extremo)
	RiskLevelLow      string = "Baixo"
	RiskLevelModerate string = "Moderado"
	RiskLevelHigh     string = "Alto"
	RiskLevelExtreme  string = "Extremo"
	// RiskLevelUndefined é usado se o cálculo não for possível
	RiskLevelUndefined string = "Indefinido"

	StatusOpen        RiskStatus = "aberto" // Mantido como está, não faz parte da solicitação de Baixo/Médio/Alto/Crítico
	StatusInProgress  RiskStatus = "em_andamento"
	StatusMitigated   RiskStatus = "mitigado"
	StatusAccepted    RiskStatus = "aceito"

	SeverityLow      VulnerabilitySeverity = "Baixo"
	SeverityMedium   VulnerabilitySeverity = "Médio"
	SeverityHigh     VulnerabilitySeverity = "Alto"
	SeverityCritical VulnerabilitySeverity = "Crítico"

	VStatusDiscovered VulnerabilityStatus = "descoberta" // Mantido como está
	VStatusInRemediation VulnerabilityStatus = "em_correcao"
	VStatusRemediated VulnerabilityStatus = "corrigida"

	ApprovalPending  ApprovalStatus = "pendente"
	ApprovalApproved ApprovalStatus = "aprovado"
	ApprovalRejected ApprovalStatus = "rejeitado"

	RoleSystemAdmin UserRole = "system_admin"
	RoleAdmin       UserRole = "admin"
	RoleManager     UserRole = "manager"
	RoleUser        UserRole = "user"

	ControlStatusConformant         AuditControlStatus = "conforme"
	ControlStatusNonConformant      AuditControlStatus = "nao_conforme"
	ControlStatusPartiallyConformant AuditControlStatus = "parcialmente_conforme"
	ControlStatusNotApplicable      AuditControlStatus = "nao_aplicavel"

	CategoryTechnological RiskCategory = "tecnologico"
	CategoryOperational   RiskCategory = "operacional"
	CategoryLegal         RiskCategory = "legal"
	// Add other categories as needed

	// C2M2 Practice Evaluation Status
	PracticeStatusNotImplemented      PracticeStatus = "not_implemented"
	PracticeStatusPartiallyImplemented PracticeStatus = "partially_implemented"
	PracticeStatusFullyImplemented    PracticeStatus = "fully_implemented"
)

type PracticeStatus string

type Organization struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;"`
	Name           string    `gorm:"size:255;not null"`
	LogoURL        string    `gorm:"size:255"`
	PrimaryColor   string    `gorm:"size:7"` // #RRGGBB
	SecondaryColor string    `gorm:"size:7"` // #RRGGBB
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Users          []User          `gorm:"foreignKey:OrganizationID"`
	Risks          []Risk          `gorm:"foreignKey:OrganizationID"`
	Vulnerabilities []Vulnerability `gorm:"foreignKey:OrganizationID"`
	AuditAssessments []AuditAssessment `gorm:"foreignKey:OrganizationID"`
	IdentityProviders []IdentityProvider `gorm:"foreignKey:OrganizationID"`
	WebhookConfigurations []WebhookConfiguration `gorm:"foreignKey:OrganizationID"`
}

func (org *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	return
}

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.NullUUID `gorm:"type:uuid;index;constraint:OnDelete:CASCADE;"`
	Name           string    `gorm:"size:255;not null"`
	Email          string    `gorm:"size:255;not null;uniqueIndex"`
	PasswordHash   string    `gorm:"size:255;not null"`
	SSOProvider    string    `gorm:"size:50"`
	SocialLoginID  string    `gorm:"size:100"`
	Role           UserRole  `gorm:"type:varchar(20);not null;default:'user'"`
	IsActive       bool      `gorm:"default:true;not null;index"` // Novo campo para status do usuário
	TOTPSecret     string    `gorm:"size:255"` // Armazenar criptografado! No DB será string.
	IsTOTPEnabled  bool      `gorm:"default:false;not null"`
	TOTPBackupCodes string   `gorm:"type:text"` // JSON array de hashes dos códigos de backup
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AuthoredRisks  []Risk `gorm:"foreignKey:OwnerID"` // Risks where this user is the owner
	ApprovalRequests []ApprovalWorkflow `gorm:"foreignKey:RequesterID"`
	ApprovalAssignments []ApprovalWorkflow `gorm:"foreignKey:ApproverID"`
	RiskStakeholders []RiskStakeholder `gorm:"foreignKey:UserID"`
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return
}

type Risk struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.UUID       `gorm:"type:uuid;not null;index;constraint:OnDelete:CASCADE;"`
	Title          string          `gorm:"size:255;not null"`
	Description    string          `gorm:"type:text"`
	Category       RiskCategory    `gorm:"type:varchar(50)"`
	Impact         RiskImpact      `gorm:"type:varchar(20)"`
	Probability    RiskProbability `gorm:"type:varchar(20)"`
	RiskLevel      string          `gorm:"type:varchar(20);default:'Indefinido'"` // Nível de Risco Calculado
	Status         RiskStatus      `gorm:"type:varchar(20);default:'aberto';index"`
	OwnerID        uuid.UUID       `gorm:"type:uuid;constraint:OnDelete:SET NULL;"` // FK to User
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Owner          User              `gorm:"foreignKey:OwnerID"` // Relação Belongs To User
	Stakeholders   []RiskStakeholder `gorm:"foreignKey:RiskID;constraint:OnDelete:CASCADE;"`
	ApprovalWorkflows []ApprovalWorkflow `gorm:"foreignKey:RiskID;constraint:OnDelete:CASCADE;"`
}

func (risk *Risk) BeforeCreate(tx *gorm.DB) (err error) {
	if risk.ID == uuid.Nil {
		risk.ID = uuid.New()
	}
	return
}

type Vulnerability struct {
	ID             uuid.UUID             `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.UUID             `gorm:"type:uuid;not null;index;constraint:OnDelete:CASCADE;"`
	Title          string                `gorm:"size:255;not null"`
	Description    string                `gorm:"type:text"`
	CVEID          string                `gorm:"size:50;index"` // Optional
	Severity       VulnerabilitySeverity `gorm:"type:varchar(20);index"`
	Status         VulnerabilityStatus   `gorm:"type:varchar(20);default:'descoberta';index"`
	AssetAffected  string                `gorm:"size:255"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (vuln *Vulnerability) BeforeCreate(tx *gorm.DB) (err error) {
	if vuln.ID == uuid.Nil {
		vuln.ID = uuid.New()
	}
	return
}

// Join table for many-to-many relationship between Risks and Users (Stakeholders)
type RiskStakeholder struct {
	RiskID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Risk      Risk      `gorm:"foreignKey:RiskID;constraint:OnDelete:CASCADE;"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	CreatedAt time.Time
}

type ApprovalWorkflow struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;"`
	RiskID      uuid.UUID      `gorm:"type:uuid;not null;constraint:OnDelete:CASCADE;"`
	RequesterID uuid.UUID      `gorm:"type:uuid;constraint:OnDelete:SET NULL;"` // FK to User
	ApproverID  uuid.UUID      `gorm:"type:uuid;constraint:OnDelete:SET NULL;"` // FK to User
	Status      ApprovalStatus `gorm:"type:varchar(20);default:'pendente'"`
	Comments    string         `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Risk        Risk           `gorm:"foreignKey:RiskID"`
	Requester   User           `gorm:"foreignKey:RequesterID"`
	Approver    User           `gorm:"foreignKey:ApproverID"`
}

func (aw *ApprovalWorkflow) BeforeCreate(tx *gorm.DB) (err error) {
	if aw.ID == uuid.Nil {
		aw.ID = uuid.New()
	}
	return
}

type AuditFramework struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;"`
	Name           string    `gorm:"size:255;not null;uniqueIndex"` // NIST CSF 2.0, CIS Controls v8, etc.
	AuditControls  []AuditControl `gorm:"foreignKey:FrameworkID;constraint:OnDelete:CASCADE;"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (af *AuditFramework) BeforeCreate(tx *gorm.DB) (err error) {
	if af.ID == uuid.Nil {
		af.ID = uuid.New()
	}
	return
}

type AuditControl struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;"`
	FrameworkID uuid.UUID `gorm:"type:uuid;not null"`
	ControlID   string    `gorm:"size:50;not null"` // e.g., AC-1, PR.IP-2
	Description string    `gorm:"type:text"`
	Family      string    `gorm:"size:100"` // e.g., Access Control, Identify
	Framework   AuditFramework `gorm:"foreignKey:FrameworkID;constraint:OnDelete:CASCADE;"` // Se o Framework for deletado, os controles também são.
	AuditAssessments []AuditAssessment `gorm:"foreignKey:AuditControlID;constraint:OnDelete:CASCADE;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeforeCreate hook to ensure ID is set
func (ac *AuditControl) BeforeCreate(tx *gorm.DB) (err error) {
	if ac.ID == uuid.Nil {
		ac.ID = uuid.New()
	}
	return
}


type AuditAssessment struct {
	ID             uuid.UUID          `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.UUID          `gorm:"type:uuid;not null;index;constraint:OnDelete:CASCADE;"`
	// Storing AuditControl's UUID for a more robust FK relationship
	AuditControlID uuid.UUID          `gorm:"type:uuid;not null;index"` // FK to AuditControl's ID
	Status         AuditControlStatus `gorm:"type:varchar(30)" json:"status"`
	EvidenceURL    string             `gorm:"size:255" json:"evidence_url"`
	Score          *int               `json:"score,omitempty"` // Integer score, ponteiro para ser omitempty
	AssessmentDate *time.Time         `gorm:"type:timestamptz" json:"assessment_date,omitempty"` // Ponteiro para ser omitempty
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`

	// Campos para Maturidade C2M2
	C2M2AssessmentDate *time.Time `gorm:"type:timestamptz" json:"c2m2_assessment_date,omitempty"` // Data da avaliação de maturidade C2M2
	C2M2Comments      *string    `gorm:"type:text" json:"c2m2_comments,omitempty"`         // Comentários da avaliação C2M2

	AuditControl   AuditControl       `gorm:"foreignKey:AuditControlID;constraint:OnDelete:CASCADE;" json:"audit_control,omitempty"` // Se o AuditControl for deletado
	// A OrganizationID também é uma FK. Se a Organization for deletada, as Assessments devem ser deletadas.
	// Isso será tratado na definição da relação em Organization struct.
	C2M2PracticeEvaluations []C2M2PracticeEvaluation `gorm:"foreignKey:AuditAssessmentID;constraint:OnDelete:CASCADE;" json:"c2m2_practice_evaluations,omitempty"`
}

func (as *AuditAssessment) BeforeCreate(tx *gorm.DB) (err error) {
	if as.ID == uuid.Nil {
		as.ID = uuid.New()
	}
	return
}

// --- C2M2 Models ---

// C2M2Domain representa um domínio do Cybersecurity Capability Maturity Model.
type C2M2Domain struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id"`
	Name      string         `gorm:"size:255;not null;uniqueIndex" json:"name"` // Ex: "Risk Management"
	Code      string         `gorm:"size:10;not null;uniqueIndex" json:"code"`  // Ex: "RM"
	Practices []C2M2Practice `gorm:"foreignKey:DomainID;constraint:OnDelete:CASCADE;" json:"practices,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (d *C2M2Domain) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

// C2M2Practice representa uma prática específica dentro de um domínio C2M2.
type C2M2Practice struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;" json:"id"`
	DomainID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"domain_id"`
	Code        string     `gorm:"size:20;not null;uniqueIndex" json:"code"`        // Ex: "RM.1.1"
	Description string     `gorm:"type:text;not null" json:"description"`
	TargetMIL   int        `gorm:"not null" json:"target_mil"`                      // Nível de maturidade alvo (1, 2, ou 3)
	Domain      C2M2Domain `gorm:"foreignKey:DomainID" json:"-"` // Omitir para evitar ciclos JSON
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (p *C2M2Practice) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

// C2M2PracticeEvaluation armazena a avaliação de uma prática específica para uma avaliação de controle.
type C2M2PracticeEvaluation struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	AuditAssessmentID uuid.UUID `gorm:"type:uuid;not null;index" json:"audit_assessment_id"`
	PracticeID        uuid.UUID `gorm:"type:uuid;not null;index" json:"practice_id"`
	Status            PracticeStatus    `gorm:"type:varchar(50);not null" json:"status"` // not_implemented, partially_implemented, fully_implemented
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (e *C2M2PracticeEvaluation) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return
}


// Helper function to initialize DB connection (example)
// This would typically be in a database package
/*
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
*/

// IdentityProviderType defines the type of identity provider
type IdentityProviderType string

const (
	IDPTypeSAML         IdentityProviderType = "saml"
	IDPTypeOAuth2Google IdentityProviderType = "oauth2_google"
	IDPTypeOAuth2Github IdentityProviderType = "oauth2_github"
	// Add other types as needed
)

// IdentityProvider stores configuration for SSO/Social Login providers
type IdentityProvider struct {
	ID                   uuid.UUID            `gorm:"type:uuid;primary_key;"`
	OrganizationID       uuid.UUID            `gorm:"type:uuid;not null;index"` // Foreign key to organizations
	ProviderType         IdentityProviderType `gorm:"type:varchar(50);not null"`
	Name                 string               `gorm:"size:100;not null"` // User-friendly name, e.g., "Login com Google Corporativo"
	IsActive             bool                 `gorm:"default:true;not null"`
	ConfigJSON           string               `gorm:"type:jsonb"` // Stores SAML URLs, OAuth2 client_id/secret, etc.
	AttributeMappingJSON string               `gorm:"type:jsonb"` // Optional: maps IdP attributes to User fields
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Organization         Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE;"`
}

// BeforeCreate hook to ensure ID is set for IdentityProvider
func (idp *IdentityProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if idp.ID == uuid.Nil {
		idp.ID = uuid.New()
	}
	return
}

// WebhookEventType define os tipos de eventos que podem disparar webhooks.
type WebhookEventType string

const (
	EventTypeRiskCreated        WebhookEventType = "risk_created"
	EventTypeRiskStatusChanged  WebhookEventType = "risk_status_changed"
	// Adicionar outros tipos de evento conforme necessário
)

// WebhookConfiguration armazena a configuração para um webhook específico.
type WebhookConfiguration struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name           string    `gorm:"size:100;not null"`
	URL            string    `gorm:"size:2048;not null"` // URL do webhook
	// EventTypes armazena uma lista de eventos que disparam este webhook.
	// Usando TEXT para simplicidade, poderia ser JSONB ou uma tabela de junção para mais estrutura.
	// Se usar array nativo do Postgres: `gorm:"type:text[]"` - requer driver compatível e setup.
	// Para JSONB: `gorm:"type:jsonb"` - armazenar como um array JSON de strings.
	EventTypes     string    `gorm:"type:text"` // Ex: "risk_created,risk_status_changed" (separado por vírgula) ou JSON array string
	IsActive       bool      `gorm:"default:true;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE;"`
}

// BeforeCreate hook para WebhookConfiguration
func (wc *WebhookConfiguration) BeforeCreate(tx *gorm.DB) (err error) {
	if wc.ID == uuid.Nil {
		wc.ID = uuid.New()
	}
	// Validação básica de URL pode ser adicionada aqui se necessário,
	// mas geralmente é melhor no handler.
	return
}


// AutoMigrateDB automatically migrates the schema
func AutoMigrateDB(db *gorm.DB) error {
	err := db.AutoMigrate(
		&Organization{},
		&User{},
		&Risk{},
		&Vulnerability{},
		&RiskStakeholder{},
		&ApprovalWorkflow{},
		&AuditFramework{},
		&AuditControl{},
		&AuditAssessment{},
		&IdentityProvider{},
		&WebhookConfiguration{},
		// C2M2 Models
		&C2M2Domain{},
		&C2M2Practice{},
		&C2M2PracticeEvaluation{},
	)
	return err
}
