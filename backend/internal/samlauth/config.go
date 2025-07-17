package samlauth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"phoenixgrc/backend/internal/models"

	"github.com/crewjam/saml/samlsp"
)

type SAMLIdPConfig struct {
	SpEntityID  string `json:"sp_entity_id"`
	SignRequest bool   `json:"sign_request"`
}

var (
	spRootURL     string
	spKey         *rsa.PrivateKey
	spCertificate *x509.Certificate
)

func InitializeSAMLSPGlobalConfig() error {
	spRootURL = os.Getenv("APP_ROOT_URL")
	if spRootURL == "" {
		return fmt.Errorf("APP_ROOT_URL environment variable not set")
	}
	spKeyPEM := os.Getenv("SAML_SP_KEY_PEM")
	spCertPEM := os.Getenv("SAML_SP_CERT_PEM")
	if spKeyPEM == "" || spCertPEM == "" {
		return fmt.Errorf("SAML_SP_KEY_PEM and SAML_SP_CERT_PEM must be set")
	}
	block, _ := pem.Decode([]byte(spKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to decode SP private key PEM")
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse SP private key: %w", err)
		}
	}
	var ok bool
	spKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("SP key is not RSA private key")
	}
	certBlock, _ := pem.Decode([]byte(spCertPEM))
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("failed to decode SP certificate PEM")
	}
	spCertificate, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse SP certificate: %w", err)
	}
	return nil
}

func GetSAMLServiceProviderOptions(idpModel *models.IdentityProvider) (*samlsp.Options, error) {
	if spKey == nil || spCertificate == nil || spRootURL == "" {
		return nil, fmt.Errorf("SAML SP global config not initialized")
	}
	var cfg SAMLIdPConfig
	// Ignorando erro de Unmarshal aqui, pois cfg será zero-value se ConfigJSON for inválido ou vazio,
	// o que é tratado por defaults abaixo. Em um cenário de produção robusto, o erro seria logado.
	_ = json.Unmarshal([]byte(idpModel.ConfigJSON), &cfg)

	parsedSpRootURL, err := url.Parse(spRootURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP root URL: %w", err)
	}

	// ACS URL deve ser construída dinamicamente baseada no ID do IdP e na raiz da aplicação.
	// Ex: https://app.example.com/auth/saml/uuid-do-idp/acs
	acsURLString := fmt.Sprintf("%s/auth/saml/%s/acs", spRootURL, idpModel.ID.String())
	acsURL, err := url.Parse(acsURLString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ACS URL '%s': %w", acsURLString, err)
	}

	// Metadata URL também deve ser dinâmica.
	// Ex: https://app.example.com/auth/saml/uuid-do-idp/metadata
	metadataURLString := fmt.Sprintf("%s/auth/saml/%s/metadata", spRootURL, idpModel.ID.String())
	// parsedMetadataURL, err := url.Parse(metadataURLString) // Não usado diretamente em samlsp.Options, mas bom para consistência
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse Metadata URL '%s': %w", metadataURLString, err)
	// }


	// SpEntityID pode vir da configuração do IdP no banco, ou default para a URL de metadados do SP.
	spEntityID := cfg.SpEntityID
	if spEntityID == "" {
		spEntityID = metadataURLString // Default EntityID para a URL de metadados do SP
	}


	opts := samlsp.Options{
		URL:         *parsedSpRootURL, // URL base da aplicação (Service Provider)
		Key:         spKey,
		Certificate: spCertificate,
		EntityID:    spEntityID,    // EntityID do Service Provider
		SignRequest: cfg.SignRequest, // Se as AuthNRequests devem ser assinadas
		// ACSURL:      *acsURL,        // Assertion Consumer Service URL
		// IDPMetadataURL: idpModel.IDPMetadataURL, // Se o metadata do IdP for buscado de uma URL
		// IDPMetadata: &idpModel.IDPMetadata, // Se o metadata do IdP for fornecido diretamente (XML)
		// ForceAuthn: false, // Se o IdP deve forçar re-autenticação do usuário
	}

	// Para usar IDPMetadata, o idpModel precisaria ter o XML dos metadados do IdP.
	// A biblioteca crewjam/saml pode buscar de uma URL se IDPMetadataURL for fornecido,
	// ou usar um objeto `EntityDescriptor` se IDPMetadata for fornecido.
	// O `config_json` do `IdentityProvider` deve conter `idp_metadata_url` ou o XML direto.
	// Exemplo: se `idpModel.ConfigJSON` contiver `{"idp_metadata_url":"http://idp.example.com/metadata"}`
	// ou `{"idp_entity_id":"...", "idp_sso_url":"...", "idp_x509_cert":"..."}`

	// A lógica atual em idp_handler.go para SAML espera "idp_entity_id", "idp_sso_url", "idp_x509_cert"
	// Isso significa que não estamos usando IDPMetadataURL ou IDPMetadata diretamente com crewjam/saml
	// para buscar/parsear metadados do IdP, mas sim configurando manualmente. Isso é aceitável.

	return &opts, nil
}
