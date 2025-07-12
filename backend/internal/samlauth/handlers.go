package samlauth

import (
	"fmt"
	"net/http"
	"net/url"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
	// "phoenixgrc/backend/internal/auth" // Para gerar token JWT da aplicação
	// "phoenixgrc/backend/internal/database" // Para buscar/criar usuário
	// "phoenixgrc/backend/pkg/config" // Para FRONTEND_SAML_CALLBACK_URL

	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getSAMLServiceProvider retrieves an IdentityProvider model from DB (TODO)
// and configures a samlsp.Middleware instance for it.
func getSAMLServiceProvider(c *gin.Context, idpID uuid.UUID) (*samlsp.Middleware, *models.IdentityProvider, error) {
	// TODO: Fetch IdP model from database using idpID
	// db := database.GetDB()
	// var idpModel models.IdentityProvider
	// if err := db.First(&idpModel, "id = ?", idpID).Error; err != nil {
	// 	 return nil, nil, fmt.Errorf("failed to find IdP with ID %s: %w", idpID, err)
	// }
	// if !idpModel.IsActive || idpModel.ProviderType != models.IDPTypeSAML {
	//	 return nil, nil, fmt.Errorf("IdP %s is not an active SAML provider", idpID)
	// }

	// ---- Placeholder for DB fetch ----
	var idpModel models.IdentityProvider
	idpModel.ID = idpID
	// ConfigJSON deve ser preenchido pelo DB. Exemplo mínimo para compilar:
	idpModel.ConfigJSON = string([]byte(`{"sp_entity_id": "phoenix-grc-sp","sign_request":false, "idp_entity_id":"dummy-idp-entity", "idp_sso_url":"http://dummy.idp/sso", "idp_x509_cert":"..."}`))
	idpModel.ProviderType = models.IDPTypeSAML
	idpModel.IsActive = true
	// ---- Fim do Placeholder ----

	opts, err := GetSAMLServiceProviderOptions(&idpModel)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to get SAML SP options for IdP %s: %w", idpID, err)
	}
	if opts == nil { // Should be caught by err check above, but defensive
		return nil, &idpModel, fmt.Errorf("SAML SP options are nil for IdP %s", idpID)
	}
	// AcsURL e MetadataURL são construídas dinamicamente em GetSAMLServiceProviderOptions

	spMiddleware, err := samlsp.New(*opts)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to create samlsp.Middleware for IdP %s: %w", idpID, err)
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

func ACSHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	middleware, idpModel, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		phxlog.L.Error("Error getting SAML SP for ACS",
			zap.String("idpID", idpIDStr),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for ACS."})
		return
	}

	// Parse a requisição SAML (Assertion)
	// O middleware.RequireAccount já faz isso e popula c.Request.Context com a asserção
	// se a autenticação SAML for bem-sucedida.
	// No entanto, para obter a asserção diretamente para provisionamento de usuário,
	// podemos precisar de uma abordagem um pouco diferente ou usar os dados já processados.

	// Para este exemplo, vamos assumir que queremos processar a asserção e, em seguida,
	// redirecionar ou emitir um token JWT da nossa aplicação.
	// A biblioteca crewjam/saml pode ser um pouco complexa aqui.
	// A forma mais simples com `samlsp.Middleware` é que ele tem seu próprio session handler.
	// Se quisermos integrar com nosso JWT, precisamos pegar os atributos da asserção.

	// O middleware.RequireAccount protege o handler. Se chegar aqui, o usuário foi autenticado pelo IdP.
	// A asserção pode ser recuperada do contexto da requisição.
	// s, err := middleware.Session.GetSession(c.Request)
	// if err != nil {
	// 	 c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get SAML session"})
	// 	 return
	// }
	// assertion := s.(samlsp.SessionWithAttributes).GetAttributes()
	// email := assertion.Get("email") // Ajustar nome do atributo conforme mapeamento

	// TODO: Implementar a lógica real do ACS:
	// 1. Validar a asserção SAML (o middleware já faz grande parte disso).
	// 2. Extrair atributos do usuário (email, nome, etc.) da asserção.
	//    O mapeamento de atributos pode ser configurado no `idpModel.AttributeMappingJSON`.
	// 3. Procurar o usuário no banco de dados pelo email ou outro identificador único.
	// 4. Se o usuário não existir, provisioná-lo (criá-lo) na organização associada ao `idpModel.OrganizationID`.
	//    Garantir que a organização exista e seja a correta.
	// 5. Se o usuário existir, atualizar seus dados se necessário.
	// 6. Gerar um token JWT da aplicação Phoenix GRC para o usuário.
	// 7. Redirecionar o usuário para o frontend com o token JWT.
	//    (ex: appCfg.Cfg.FrontendBaseURL + "/auth/saml/callback?token=" + jwtToken)

	phxlog.L.Info("SAML ACSHandler invoked - Placeholder for full implementation.",
		zap.String("idpID", idpModel.ID.String()),
		zap.String("idpName", idpModel.Name))
	// Exemplo de como obter atributos (requer que o middleware já tenha processado):
	// samlSession, _ := middleware.Session.GetSession(c.Request)
	// if samlSession != nil {
	//    attrs := samlSession.(samlsp.SessionWithAttributes).GetAttributes()
	//    phxlog.L.Debug("SAML Attributes received", zap.Any("attributes", attrs))
	// }

	// Tentativa de obter a sessão SAML (que conteria a asserção processada pelo middleware)
	// O middleware.RequireAccount deveria ter sido aplicado a esta rota para que isto funcione.
	// Se não, c.Request.Context() pode não ter a asserção.
	// Por agora, o foco é no placeholder.
	var samlAssertionAttributes samlsp.Attributes
	session, errSession := middleware.Session.GetSession(c.Request)
	if errSession == nil && session != nil {
		if s, ok := session.(samlsp.SessionWithAttributes); ok {
			samlAssertionAttributes = s.GetAttributes()
			phxlog.L.Info("SAML assertion attributes received by ACS (placeholder)",
				zap.String("idpID", idpModel.ID.String()),
				zap.String("idpName", idpModel.Name),
				zap.Any("attributes", samlAssertionAttributes),
			)
		}
	} else if errSession != nil {
		phxlog.L.Warn("Could not get SAML session in ACS handler", zap.Error(errSession), zap.String("idpID", idpModel.ID.String()))
	}


	c.JSON(http.StatusNotImplemented, gin.H{
		"message":                       "SAML ACS logic is partially implemented. Assertion may have been received by SP.",
		"status":                        "pending_full_user_processing_and_jwt_issuance",
		"idp_id":                        idpModel.ID.String(),
		"idp_name":                      idpModel.Name,
		"received_attributes_example":   samlAssertionAttributes, // Pode ser nil
		"next_steps":                    "Backend needs to fully process assertion, provision user, and issue Phoenix GRC JWT.",
	})
	// c.String(http.StatusNotImplemented, "SAML ACS Handler for IdP %s (IdP Name: %s) - Not Fully Implemented. Assertion received, but processing logic is pending. User attributes would be extracted here, user provisioned/updated, and a session/JWT for Phoenix GRC would be issued, followed by a redirect to the frontend.", idpModel.ID, idpModel.Name)
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
	authReq, err := middleware.ServiceProvider.MakeAuthenticationRequest(middleware.ServiceProvider.GetSSOBindingLocation(samlsp.HTTPRedirectBinding))
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
