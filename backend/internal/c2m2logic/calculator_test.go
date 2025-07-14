package c2m2logic

import (
	"log"
	"os"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mockDB *gorm.DB
var sqlMock sqlmock.Sqlmock

func setupTestDB(t *testing.T) {
	var err error
	db, smock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	sqlMock = smock

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent, // Change to logger.Info for SQL logs
			Colorful:      true,
		},
	)
	mockDB, err = gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: gormLogger})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = mockDB
}

func TestCalculateAndUpdateMaturityLevel(t *testing.T) {
	setupTestDB(t)

	assessmentID := uuid.New()

	// --- Mock de todas as práticas C2M2 no DB ---
	practice1_mil1 := models.C2M2Practice{ID: uuid.New(), Code: "RM.1.1", TargetMIL: 1}
	practice2_mil1 := models.C2M2Practice{ID: uuid.New(), Code: "SA.1.1", TargetMIL: 1}
	practice1_mil2 := models.C2M2Practice{ID: uuid.New(), Code: "RM.2.1", TargetMIL: 2}
	practice2_mil2 := models.C2M2Practice{ID: uuid.New(), Code: "TVM.2.1", TargetMIL: 2}
	practice1_mil3 := models.C2M2Practice{ID: uuid.New(), Code: "RM.3.1", TargetMIL: 3}
	allPractices := []models.C2M2Practice{practice1_mil1, practice2_mil1, practice1_mil2, practice2_mil2, practice1_mil3}

	practiceRows := sqlmock.NewRows([]string{"id", "code", "target_mil"})
	for _, p := range allPractices {
		practiceRows.AddRow(p.ID, p.Code, p.TargetMIL)
	}


	testCases := []struct {
		name          string
		evaluations   []models.C2M2PracticeEvaluation // Avaliações para este assessment
		expectedMIL   int
	}{
		{
			name: "Should achieve MIL 0 - one MIL 1 practice not implemented",
			evaluations: []models.C2M2PracticeEvaluation{
				{PracticeID: practice1_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil1.ID, Status: models.PracticeStatusPartiallyImplemented},
			},
			expectedMIL: 0,
		},
		{
			name: "Should achieve MIL 1 - all MIL 1 practices fully implemented",
			evaluations: []models.C2M2PracticeEvaluation{
				{PracticeID: practice1_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice1_mil2.ID, Status: models.PracticeStatusPartiallyImplemented}, // MIL 2 practice not fully implemented
			},
			expectedMIL: 1,
		},
		{
			name: "Should achieve MIL 2 - all MIL 1 & 2 practices fully implemented",
			evaluations: []models.C2M2PracticeEvaluation{
				{PracticeID: practice1_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice1_mil2.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil2.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice1_mil3.ID, Status: models.PracticeStatusNotImplemented}, // MIL 3 practice not done
			},
			expectedMIL: 2,
		},
		{
			name: "Should achieve MIL 3 - all MIL 1, 2 & 3 practices fully implemented",
			evaluations: []models.C2M2PracticeEvaluation{
				{PracticeID: practice1_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil1.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice1_mil2.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice2_mil2.ID, Status: models.PracticeStatusFullyImplemented},
				{PracticeID: practice1_mil3.ID, Status: models.PracticeStatusFullyImplemented},
			},
			expectedMIL: 3,
		},
		{
			name: "Should achieve MIL 0 - no evaluations provided",
			evaluations: []models.C2M2PracticeEvaluation{},
			expectedMIL: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Redefinir mocks para cada caso de teste
			sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2_m2_practices"`)).WillReturnRows(practiceRows)

			evalRows := sqlmock.NewRows([]string{"practice_id", "status"})
			for _, e := range tc.evaluations {
				evalRows.AddRow(e.PracticeID, e.Status)
			}
			sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_practice_evaluations" WHERE audit_assessment_id = $1`)).
				WithArgs(assessmentID).
				WillReturnRows(evalRows)

			sqlMock.ExpectBegin()
			sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "audit_assessments" SET "c2m2_maturity_level"=$1,"updated_at"=$2 WHERE id = $3`)).
				WithArgs(tc.expectedMIL, sqlmock.AnyArg(), assessmentID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			sqlMock.ExpectCommit()


			calculatedMIL, err := CalculateAndUpdateMaturityLevel(assessmentID, mockDB)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedMIL, calculatedMIL)
			assert.NoError(t, sqlMock.ExpectationsWereMet())
		})
	}
}
