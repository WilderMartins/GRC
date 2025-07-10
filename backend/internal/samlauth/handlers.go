package samlauth

// TODO: JULES - Pacote SAMLauth temporariamente comentado devido a problemas de compilação persistentes
// com a biblioteca github.com/crewjam/saml no ambiente de sandbox.
// A funcionalidade SAML precisará ser reativada e depurada quando o problema ambiental/linker for resolvido.

/*
import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"phoenixgrc/backend/internal/models"

	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getSAMLServiceProvider(c *gin.Context, idpID uuid.UUID) (*samlsp.Middleware, *models.IdentityProvider, error) {
	var idpModel models.IdentityProvider
	idpModel.ID = idpID
	idpModel.ConfigJSON = string([]byte(`{"sp_entity_id": "test-sp-entity-id-compile-test"}`))

	opts, err := GetSAMLServiceProviderOptions(&idpModel)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("simplified GetSAMLServiceProviderOptions failed: %w", err)
	}
	if opts == nil {
		return nil, &idpModel, fmt.Errorf("SAML SP options are nil (simplified)")
	}
	if opts.AcsURL == nil {
		log.Println("Warning: opts.AcsURL was nil in getSAMLServiceProvider, creating a dummy one.")
		dummyAcsURL := fmt.Sprintf("http://localhost:8080/auth/saml/%s/acs", idpID.String())
		opts.AcsURL, _ = url.Parse(dummyAcsURL)
	}

	spMiddleware, err := samlsp.New(*opts)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("simplified samlsp.New failed: %w", err)
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
		fmt.Printf("Error getting SAML SP for metadata (IdP ID: %s): %v\n", idpIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider."})
		return
	}
	middleware.ServeMetadata(c.Writer, c.Request)
}

func ACSHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	_, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	log.Printf("SAML ACSHandler for IdP %s (simplified - compile test)", idpIDStr)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "SAML ACS logic temporarily disabled for compile testing."})
}

func SAMLLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	_, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}
	log.Printf("SAML SAMLLoginHandler for IdP %s (simplified - compile test)", idpIDStr)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "SAML Login logic temporarily disabled for compile testing."})
}
*/
