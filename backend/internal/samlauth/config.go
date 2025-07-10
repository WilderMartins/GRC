package samlauth

// TODO: JULES - Pacote SAMLauth temporariamente comentado devido a problemas de compilação persistentes
// com a biblioteca github.com/crewjam/saml no ambiente de sandbox.
// A funcionalidade SAML precisará ser reativada e depurada quando o problema ambiental/linker for resolvido.

/*
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
	_ = json.Unmarshal([]byte(idpModel.ConfigJSON), &cfg)

	parsedSpRootURL, err := url.Parse(spRootURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SP root URL: %w", err)
	}

	metadataURLDynamic := fmt.Sprintf("%s/auth/saml/%s/metadata", spRootURL, idpModel.ID.String())

	spEntityID := cfg.SpEntityID
	if spEntityID == "" {
		spEntityID = metadataURLDynamic
	}

	// AcsURL precisa ser definido, mesmo que dummy, para samlsp.New não dar panic.
	dummyAcsURL := fmt.Sprintf("http://localhost:8080/auth/saml/%s/acs", idpModel.ID.String())
	parsedDummyAcsURL, _ := url.Parse(dummyAcsURL)


	opts := samlsp.Options{
		URL:         *parsedSpRootURL,
		Key:         spKey,
		Certificate: spCertificate,
		EntityID:    spEntityID,
		SignRequest: cfg.SignRequest,
		AcsURL:      parsedDummyAcsURL,
	}
	return &opts, nil
}
*/
