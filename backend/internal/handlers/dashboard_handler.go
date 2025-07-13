package handlers

import (
	"fmt"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RiskMatrixData defines the structure for the risk matrix data.
type RiskMatrixData struct {
	Probability models.RiskProbability `json:"probability"`
	Impact      models.RiskImpact      `json:"impact"`
	Count       int64                  `json:"count"`
}

// GetRiskMatrixHandler handles fetching data for the risk matrix.
func GetRiskMatrixHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()
	var results []RiskMatrixData
	if err := db.Model(&models.Risk{}).
		Select("probability, impact, count(*) as count").
		Where("organization_id = ?", organizationID).
		Group("probability, impact").
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk matrix data"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// RecentActivityData defines the structure for the recent activity feed.
type RecentActivityData struct {
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	Link      string    `json:"link"`
}

// GetRecentActivityHandler handles fetching data for the recent activity feed.
func GetRecentActivityHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()
	var activities []RecentActivityData

	// Fetch recent risks
	var risks []models.Risk
	if err := db.Where("organization_id = ?", organizationID).Order("created_at desc").Limit(5).Find(&risks).Error; err == nil {
		for _, r := range risks {
			activities = append(activities, RecentActivityData{
				Type:      "Risco",
				Title:     r.Title,
				Timestamp: r.CreatedAt,
				Link:      fmt.Sprintf("/admin/risks/%s", r.ID),
			})
		}
	}

	// Fetch recent vulnerabilities
	var vulnerabilities []models.Vulnerability
	if err := db.Where("organization_id = ?", organizationID).Order("created_at desc").Limit(5).Find(&vulnerabilities).Error; err == nil {
		for _, v := range vulnerabilities {
			activities = append(activities, RecentActivityData{
				Type:      "Vulnerabilidade",
				Title:     v.Title,
				Timestamp: v.CreatedAt,
				Link:      fmt.Sprintf("/admin/vulnerabilities/edit/%s", v.ID),
			})
		}
	}

	// Sort activities by timestamp desc
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

	// Limit to the last 10 activities overall
	if len(activities) > 10 {
		activities = activities[:10]
	}

	c.JSON(http.StatusOK, activities)
}

// VulnerabilitySummaryData defines the structure for the vulnerability summary data.
type VulnerabilitySummaryData struct {
	BySeverity []struct {
		Severity models.VulnerabilitySeverity `json:"severity"`
		Count    int64                        `json:"count"`
	} `json:"by_severity"`
	ByStatus []struct {
		Status models.VulnerabilityStatus `json:"status"`
		Count  int64                      `json:"count"`
	} `json:"by_status"`
}

// GetVulnerabilitySummaryHandler handles fetching data for the vulnerability summary.
func GetVulnerabilitySummaryHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()
	var summary VulnerabilitySummaryData

	// Summary by severity
	if err := db.Model(&models.Vulnerability{}).
		Select("severity, count(*) as count").
		Where("organization_id = ?", organizationID).
		Group("severity").
		Scan(&summary.BySeverity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vulnerability summary by severity"})
		return
	}

	// Summary by status
	if err := db.Model(&models.Vulnerability{}).
		Select("status, count(*) as count").
		Where("organization_id = ?", organizationID).
		Group("status").
		Scan(&summary.ByStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vulnerability summary by status"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// ComplianceOverviewData defines the structure for the compliance overview data.
type ComplianceOverviewData struct {
	FrameworkName string  `json:"framework_name"`
	Score         float64 `json:"score"`
}

// GetComplianceOverviewHandler handles fetching data for the compliance overview.
func GetComplianceOverviewHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()
	var results []ComplianceOverviewData

	// This query is a bit more complex. It calculates the average score per framework.
	if err := db.Table("audit_frameworks").
		Select("audit_frameworks.name as framework_name, COALESCE(AVG(audit_assessments.score), 0) as score").
		Joins("LEFT JOIN audit_controls ON audit_controls.framework_id = audit_frameworks.id").
		Joins("LEFT JOIN audit_assessments ON audit_assessments.audit_control_id = audit_controls.id AND audit_assessments.organization_id = ?", organizationID).
		Group("audit_frameworks.id").
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch compliance overview data"})
		return
	}

	c.JSON(http.StatusOK, results)
}
