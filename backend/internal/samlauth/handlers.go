package samlauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/pkg/config"
	phxlog "phoenixgrc/backend/pkg/log"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// getSAMLServiceProvider retrieves an IdentityProvider model from DB
// and configures a samlsp.Middleware instance for it.
func getSAMLServiceProvider(c *gin.Context, idpID uuid.UUID) (*samlsp.Middleware, *models.IdentityProvider, error) {
	db := database.GetDB() // Obter instância do DB
	var idpModel models.IdentityProvider

	// Buscar o IdentityProvider no banco de dados
	if err := db.First(&idpModel, "id = ?", idpID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, fmt.Errorf("SAML Identity Provider with ID %s not found", idpID)
		}
		return nil, nil, fmt.Errorf("failed to query SAML Identity Provider with ID %s: %w", idpID, err)
	}

	// Validar se o IdP está ativo e é do tipo SAML
	if !idpModel.IsActive {
		return nil, &idpModel, fmt.Errorf("SAML Identity Provider %s (Name: %s) is not active", idpID, idpModel.Name)
	}
	if idpModel.ProviderType != models.IDPTypeSAML {
		return nil, &idpModel, fmt.Errorf("Identity Provider %s (Name: %s) is not a SAML provider (Type: %s)", idpID, idpModel.Name, idpModel.ProviderType)
	}

	// ConfigJSON já deve estar preenchido pelo GORM a partir do DB.

	opts, err := GetSAMLServiceProviderOptions(&idpModel)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to get SAML SP options for IdP %s (Name: %s): %w", idpID, idpModel.Name, err)
	}
	if opts == nil { // Defensivo, GetSAMLServiceProviderOptions deve retornar erro se opts for nil
		return nil, &idpModel, fmt.Errorf("SAML SP options are nil for IdP %s (Name: %s)", idpID, idpModel.Name)
	}

	spMiddleware, err := samlsp.New(*opts)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to create samlsp.Middleware for IdP %s (Name: %s): %w", idpID, idpModel.Name, err)
	}
	return spMiddleware, &idpModel, nil
}

func MetadataHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	middleware, _, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		phxlog.L.Error("Error getting SAML SP for metadata",
			zap.String("idpID", idpIDStr),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for metadata."})
		return
	}
	middleware.ServeMetadata(c.Writer, c.Request)
}


// ACSAttributeMapping define a estrutura esperada para o AttributeMappingJSON
type ACSAttributeMapping struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	// Adicionar outros campos conforme necessário (ex: NameID, Role/Groups)
}


// --- Implementação do ACSHandler ---
// O ACSHandler processa a SAMLResponse enviada pelo IdP.
func ACSHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		phxlog.L.Warn("Invalid IdP ID format in ACS request", zap.String("idpIDStr", idpIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	spMiddleware, idpModel, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		phxlog.L.Error("Error getting SAML SP for ACS processing", zap.String("idpID", idpIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for ACS."})
		return
	}

	// Fazer o middleware SAML processar a requisição.
	// Isso validará a SAMLResponse e, se bem-sucedido, criará uma sessão SAML.
	spMiddleware.ServeHTTP(c.Writer, c.Request)

	// Verificar se o middleware já escreveu uma resposta (ex: em caso de erro SAML)
	if c.Writer.Written() {
		phxlog.L.Info("SAML middleware handled the response directly (e.g. error or redirect). ACS processing finished by middleware.",
			zap.String("idpID", idpIDStr),
			zap.Int("status", c.Writer.Status()))
		return
	}

	// Se o middleware não escreveu uma resposta, a asserção foi válida e uma sessão foi criada.
	// Obter a sessão e os atributos.
	s, err := spMiddleware.Session.GetSession(c.Request)
	if err != nil {
		phxlog.L.Error("Failed to get SAML session after middleware processing (expected session)",
			zap.String("idpID", idpIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve SAML session after successful assertion."})
		return
	}
	if s == nil {
		phxlog.L.Error("SAML session is nil after middleware processing (expected session)", zap.String("idpID", idpIDStr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SAML session not found after successful assertion."})
		return
	}

	// Extrair atributos da asserção. O tipo concreto da sessão é JWTSessionClaims.
	samlSession, ok := s.(samlsp.JWTSessionClaims)
	if !ok {
		phxlog.L.Error("SAML session is not of expected type JWTSessionClaims", zap.String("idpID", idpIDStr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SAML session attributes not available in the expected format."})
		return
	}
	attrs := samlSession.GetAttributes()
	nameID := samlSession.Subject // O NameID está no campo Subject das claims do JWT.

	// Parsear o mapeamento de atributos do IdP
	var attrMapping ACSAttributeMapping
	if idpModel.AttributeMappingJSON != "" {
		if err := json.Unmarshal([]byte(idpModel.AttributeMappingJSON), &attrMapping); err != nil {
			phxlog.L.Error("Failed to parse AttributeMappingJSON for SAML IdP",
				zap.String("idpID", idpIDStr), zap.String("idpName", idpModel.Name), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing IdP attribute mapping."})
			return
		}
	} else {
		// Mapeamentos padrão se não configurado (ajustar conforme necessidade)
		attrMapping.Email = "email" // Ou "mail", "EmailAddress", etc.
		attrMapping.FirstName = "firstName" // Ou "givenName"
		attrMapping.LastName = "lastName"   // Ou "sn", "surname"
	}

	email := strings.ToLower(strings.TrimSpace(attrs.Get(attrMapping.Email)))
	if email == "" {
		phxlog.L.Warn("Email attribute not found or empty in SAML assertion",
			zap.String("idpID", idpIDStr), zap.String("idpName", idpModel.Name),
			zap.String("expectedEmailAttribute", attrMapping.Email), zap.Any("allAttributes", attrs))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email attribute missing or empty in SAML assertion."})
		return
	}

	firstName := strings.TrimSpace(attrs.Get(attrMapping.FirstName))
	lastName := strings.TrimSpace(attrs.Get(attrMapping.LastName))
	fullName := strings.TrimSpace(firstName + " " + lastName)
	if fullName == "" {
		fullName = email // Fallback para nome completo
	}

	// --- User Provisioning/Login ---
	db := database.GetDB()
	var user models.User
	err = db.Where("email = ? AND organization_id = ?", email, idpModel.OrganizationID).First(&user).Error

	if err == gorm.ErrRecordNotFound { // Usuário não existe, provisionar
		var allowUserCreation string
		allowUserCreation, err = models.GetSystemSetting(db, "ALLOW_SAML_USER_CREATION")
		if err != nil {
			// Se a configuração não existir, assuma um padrão seguro (não permitir criação)
			allowUserCreation = "false"
		}

		if allowUserCreation != "true" {
			phxlog.L.Warn("SAML user creation disabled, user not provisioned",
				zap.String("email", email), zap.String("idpName", idpModel.Name))
			c.JSON(http.StatusForbidden, gin.H{"error": "New user registration via this SAML provider is disabled."})
			return
		}

		user = models.User{
			OrganizationID: uuid.NullUUID{UUID: idpModel.OrganizationID, Valid: true},
			Name:           fullName,
			Email:          email,
			PasswordHash:   "SAML_USER_NO_PASSWORD_" + uuid.New().String(), // Senha não usada, mas campo é NOT NULL
			SSOProvider:    idpModel.Name,
			SocialLoginID:  nameID, // Usar NameID da asserção SAML
			Role:           models.RoleUser, // Ou buscar de um atributo SAML mapeado para role
			IsActive:       true,
		}
		if createErr := db.Create(&user).Error; createErr != nil {
			phxlog.L.Error("Failed to create new SAML user",
				zap.String("email", email), zap.String("idpName", idpModel.Name), zap.Error(createErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to provision SAML user."})
			return
		}
		phxlog.L.Info("New SAML user provisioned",
			zap.String("userID", user.ID.String()), zap.String("email", email), zap.String("idpName", idpModel.Name))
	} else if err != nil { // Outro erro de DB
		phxlog.L.Error("Database error fetching user for SAML login",
			zap.String("email", email), zap.String("idpName", idpModel.Name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during SAML login."})
		return
	} else { // Usuário existe
		user.SSOProvider = idpModel.Name
		user.SocialLoginID = nameID
		user.IsActive = true // Garantir que esteja ativo
		// Opcional: Atualizar nome se mudou no IdP
		if user.Name == "" || user.Name == user.Email { // Só atualiza se nome atual for placeholder
			user.Name = fullName
		}
		if saveErr := db.Save(&user).Error; saveErr != nil {
			phxlog.L.Error("Failed to update existing SAML user",
				zap.String("userID", user.ID.String()), zap.String("email", email), zap.Error(saveErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user during SAML login."})
			return
		}
		phxlog.L.Info("Existing SAML user logged in and updated",
			zap.String("userID", user.ID.String()), zap.String("email", email))
	}

	// Gerar token JWT da aplicação
	appToken, err := auth.GenerateToken(&user, user.OrganizationID.UUID)
	if err != nil {
		phxlog.L.Error("Failed to generate application token after SAML login",
			zap.String("userID", user.ID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token."})
		return
	}

	// Redirecionar para o frontend com o token
	// O frontend precisa ter uma rota /saml/callback para processar este token
	frontendSAMLCallbackURL := strings.TrimSuffix(config.Cfg.FrontendBaseURL, "/") + "/saml/callback"
	targetURL := fmt.Sprintf("%s?token=%s&sso_success=true&provider=saml&idp_name=%s",
		frontendSAMLCallbackURL,
		url.QueryEscape(appToken),
		url.QueryEscape(idpModel.Name),
	)
	phxlog.L.Info("SAML login successful, redirecting to frontend",
		zap.String("userID", user.ID.String()),
		zap.String("redirectURL", targetURL))
	c.Redirect(http.StatusFound, targetURL)
}

func SAMLLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	middleware, idpModel, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		idpName := "N/A"
		if idpModel != nil { // idpModel pode ser nil se getSAMLServiceProvider falhar muito cedo
			idpName = idpModel.Name
		}
		phxlog.L.Error("Error getting SAML SP for Login",
			zap.String("idpID", idpIDStr),
			zap.String("idpName", idpName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for Login."})
		return
	}

	// middleware.HandleStartAuthFlow é usado para iniciar o fluxo de login.
	// Ele redirecionará o usuário para o IdP.
	// O `relayState` pode ser usado para passar informações de volta para o ACS,
	// como a URL para a qual redirecionar após o login bem-sucedido no SP.
	// Por enquanto, vamos manter simples.
	relayState := c.Query("redirect_url") // Opcional: pegar de query param
	if relayState == "" {
		// frontendCallback := os.Getenv("FRONTEND_SAML_CALLBACK_URL") // Deveria vir da config
		// if frontendCallback == "" { frontendCallback = "/" }
		// relayState = frontendCallback
		// Vamos deixar o relayState vazio por enquanto, o middleware pode ter um default.
	}

	// A função RequireAccount já inicia o fluxo se não houver sessão.
	// Para um link de login explícito (SP-initiated), podemos precisar chamar algo como
	// middleware.HandleAuthnRequest(c.Writer, c.Request) ou similar,
	// mas a forma como crewjam/saml é usado com gin pode ser via middleware.ServeHTTP
	// ou protegendo um handler com middleware.RequireAccount.

	// Para SP-initiated login, o middleware geralmente tem um endpoint /saml/login
	// que, quando acessado, redireciona para o IdP.
	// Aqui, estamos fazendo isso manualmente.
	// A biblioteca `samlsp` espera que você use `middleware.RequireAccount` para proteger um handler.
	// Se o usuário não estiver autenticado, `RequireAccount` chama `HandleStartAuthFlow`.
	// Para um link de login direto, podemos precisar construir o AuthnRequest e redirecionar.
	// No entanto, `samlsp.Middleware` já expõe `HandleStartAuthFlow`.

	// Esta é uma forma de forçar o início do fluxo.
	// O middleware.ServiceProvider.HandleStartAuthFlow(c.Writer, c.Request) pode ser o que queremos.
	// Ou, se o middleware estiver configurado para um path específico, redirecionar para ele.
	// A maneira mais simples é garantir que o middleware esteja aplicado a uma rota
	// e o acesso a essa rota (sem sessão SAML) iniciará o fluxo.
	// Se esta rota é o ponto de entrada, precisamos garantir que o middleware faça o redirect.

	// O middleware.HandleStartAuthFlow espera ser chamado quando o usuário
	// tenta acessar um recurso protegido.
	// Se esta rota é o "botão de login SAML", então precisamos iniciar o fluxo.
	// Veja `middleware.ServiceProvider.MakeAuthenticationRequest` e `Redirect`
	// ou simplesmente deixe o middleware.RequireAccount fazer seu trabalho em uma rota protegida.

	// Para um handler de login explícito como este:
	// 1. Gerar o AuthnRequest.
	// 2. Redirecionar o usuário para o IdP com o AuthnRequest.
	// O middleware.HandleStartAuthFlow faz isso.
	// No entanto, ele é um http.Handler, não uma função que chamamos diretamente com (w,r) e depois retornamos.
	// Se usarmos gin, o middleware é aplicado a um grupo de rotas ou rota específica.
	// Para este handler, precisamos simular o que o middleware faria.

	// Uma forma mais direta com crewjam/saml pode ser usar o middleware.ServiceProvider:
	authReq, err := middleware.ServiceProvider.MakeAuthenticationRequest(middleware.ServiceProvider.GetSSOBindingLocation(saml.HTTPRedirectBinding), "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SAML AuthnRequest"})
		return
	}

	// Para binding de redirecionamento:
	redirectURL, err := authReq.Redirect(relayState, &middleware.ServiceProvider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get SAML redirect URL"})
		return
	}
	c.Redirect(http.StatusFound, redirectURL.String())
}
