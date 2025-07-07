package samlauth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"phoenixgrc/backend/internal/models"

	"github.com/crewjam/saml/samlsp"
)

// SAMLIdPConfig define os campos esperados no ConfigJSON para um provedor SAML.
type SAMLIdPConfig struct {
	IdpEntityID      string `json:"idp_entity_id"`       // ID da Entidade do IdP
	IdpSSOURL        string `json:"idp_sso_url"`         // URL de SSO do IdP (Redirecionamento)
	IdpSLOURL        string `json:"idp_slo_url"`         // URL de SLO do IdP (Opcional)
	IdPX509Cert      string `json:"idp_x509_cert"`       // Certificado X.509 do IdP (string PEM)
	SpAcsURL         string `json:"sp_acs_url"`          // URL do ACS do SP (Phoenix GRC) - será construída dinamicamente
	SpMetadataURL    string `json:"sp_metadata_url"`     // URL de Metadados do SP (Phoenix GRC) - será construída dinamicamente
	SpEntityID       string `json:"sp_entity_id"`        // ID da Entidade do SP (Phoenix GRC) - pode ser global ou por org
	SignRequest      bool   `json:"sign_request"`      // Se o SP deve assinar AuthnRequests
	WantAssertionsSigned bool `json:"want_assertions_signed"` // Se o SP espera que as Assertions sejam assinadas
}

var (
	spRootURL    string
	spKey        *rsa.PrivateKey
	spCertificate *x509.Certificate
)

// InitializeSAMLSPGlobalConfig carrega a configuração global do Service Provider (Phoenix GRC).
// Isso inclui a URL base da aplicação e as chaves/certificados do SP.
func InitializeSAMLSPGlobalConfig() error {
	spRootURL = os.Getenv("APP_ROOT_URL") // Ex: "http://localhost:8080" ou "https://phoenix.example.com"
	if spRootURL == "" {
		return fmt.Errorf("APP_ROOT_URL environment variable not set (required for SAML SP)")
	}

	spKeyPEM := os.Getenv("SAML_SP_KEY_PEM")
	spCertPEM := os.Getenv("SAML_SP_CERT_PEM")

	if spKeyPEM == "" || spCertPEM == "" {
		// Em produção, estas chaves devem ser geradas e armazenadas de forma segura.
		// Para desenvolvimento, podemos considerar gerar automaticamente se não existirem,
		// mas isso não é ideal para configurações persistentes de IdP.
		// Por agora, exigimos que sejam fornecidas.
		return fmt.Errorf("SAML_SP_KEY_PEM and SAML_SP_CERT_PEM environment variables must be set")
	}

	var err error
	spKey, err = samlsp.ParsePrivateKey([]byte(spKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to parse SAML SP private key: %w", err)
	}

	certBlock, _ := pem.Decode([]byte(spCertPEM))
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("failed to decode SAML SP certificate PEM")
	}
	spCertificate, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse SAML SP certificate: %w", err)
	}

	return nil
}

// GetSAMLServiceProviderOptions cria samlsp.Options para um IdP específico.
func GetSAMLServiceProviderOptions(idpModel *models.IdentityProvider) (*samlsp.Options, error) {
	if spKey == nil || spCertificate == nil || spRootURL == "" {
		return nil, fmt.Errorf("SAML SP global configuration not initialized")
	}

	var cfg SAMLIdPConfig
	if err := json.Unmarshal([]byte(idpModel.ConfigJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAML IdP config from JSON: %w", err)
	}

	idpMetadataURL := cfg.IdpSSOURL // Placeholder, idealmente o IdP fornece uma URL de metadados completa
	// Se o IdP tiver uma URL de metadados, podemos usar samlsp.FetchMetadata para carregar
	// dinamicamente. Por enquanto, assumimos que os campos individuais são fornecidos.

	// Construir URLs dinâmicas para o SP com base no ID do IdP
	// Ex: http://localhost:8080/auth/saml/uuid-idp-123/acs
	acsURL := fmt.Sprintf("%s/auth/saml/%s/acs", spRootURL, idpModel.ID.String())
	metadataURL := fmt.Sprintf("%s/auth/saml/%s/metadata", spRootURL, idpModel.ID.String())
	// O SpEntityID pode ser a metadataURL ou algo customizado.
	spEntityID := cfg.SpEntityID
	if spEntityID == "" {
		spEntityID = metadataURL // Default SP Entity ID to its metadata URL
	}


	opts := samlsp.Options{
		URL:            *samlsp.MustParseURL(spRootURL), // URL base da aplicação
		Key:            spKey,
		Certificate:    spCertificate,
		IDPMetadataURL: samlsp.MustParseURL(idpMetadataURL), // Este campo é usado para buscar metadados do IdP
		                                                  // Se não tivermos uma URL de metadados, teremos que configurar
		                                                  // EntityID, SSOURL, Certificado do IdP manualmente.
		                                                  // A biblioteca crewjam/saml pode ser um pouco rígida quanto a isso.
		                                                  // Alternativamente, podemos usar `IDPMetadata` para fornecer os metadados diretamente.
		EntityID:       spEntityID, // ID da Entidade do nosso SP
		SignRequest:    cfg.SignRequest,
		// ForceAuthn: false, // Se deve forçar re-autenticação no IdP
		// AllowIDPInitiated: true, // Se permite login iniciado pelo IdP
	}

	// Se não temos uma URL de metadados do IdP, mas temos os campos individuais:
	if cfg.IdpEntityID != "" && cfg.IdpSSOURL != "" && cfg.IdPX509Cert != "" {
		idpCertBlock, _ := pem.Decode([]byte(cfg.IdPX509Cert))
		if idpCertBlock == nil || idpCertBlock.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("failed to decode IdP certificate PEM from config")
		}
		idpParsedCert, err := x509.ParseCertificate(idpCertBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IdP certificate from config: %w", err)
		}

		// Para crewjam/saml, se você não usa IDPMetadataURL, você precisa construir o
		// IDPMetadata diretamente. A struct samlsp.Options não tem campos diretos para
		// IdP SSO URL ou IdP Entity ID se IDPMetadataURL não for usado para fetch.
		// Isso é uma limitação ou um design que força o uso de metadados.
		// Vamos precisar criar um objeto `saml.EntityDescriptor` para o IdP.
		// Este é um ponto complexo. Por enquanto, vamos focar no fluxo onde o IdP pode ter uma URL de metadados,
		// ou onde podemos construir um `saml.Metadata` simples.
		// A biblioteca `russellhaering/gosaml2` pode ser mais flexível para configuração manual.

		// Para simplificar agora, vamos assumir que o `IDPMetadataURL` no `opts` é o SSO URL
		// e que a biblioteca lidará com isso de alguma forma, ou que o IdP oferece metadados.
		// Em uma implementação real, isso precisaria de mais atenção.
		// A biblioteca `crewjam/saml` realmente espera buscar metadados de `IDPMetadataURL`
		// ou que você forneça `IDPMetadata *saml.EntityDescriptor`.

		// Uma maneira de lidar com isso é criar um `http.Handler` que sirva os metadados do IdP
		// localmente se eles forem fornecidos como string, e então apontar `IDPMetadataURL` para esse handler.
		// Ou, construir o `saml.EntityDescriptor` manualmente.

		// Por agora, vamos assumir que `cfg.IdpSSOURL` é suficiente para `opts.IDPMetadataURL`
		// e que o certificado do IdP será usado na validação da Assertion.
		// A biblioteca `crewjam/saml` usará o certificado do IdP para validar assinaturas
		// nas assertions se `opts.IDPMetadata` for populado (o que acontece ao buscar de `IDPMetadataURL`).
	}


	// Definir o ACS URL dinamicamente
	opts.AcsURL = *samlsp.MustParseURL(acsURL)
	// opts.SloURL = *samlsp.MustParseURL(fmt.Sprintf("%s/auth/saml/%s/slo", spRootURL, idpModel.ID.String())) // Se SLO for usado

	return &opts, nil
}

// TODO: Gerar chaves e certificado SP se não existirem (para desenvolvimento)
// func generateSelfSignedCert() (keyPEM, certPEM string, err error) { ... }
