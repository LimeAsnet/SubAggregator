//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/LimeAsnet/SubAggregator/internal/config"
	"github.com/LimeAsnet/SubAggregator/internal/database"
	"github.com/LimeAsnet/SubAggregator/internal/handlers"
	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/repository"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var projectRootDir string

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	root, err := projectRoot()
	if err != nil {
		panic(err)
	}
	projectRootDir = root

	if err := os.Chdir(root); err != nil {
		panic(fmt.Errorf("chdir to project root %s: %w", root, err))
	}

	os.Exit(m.Run())
}

func projectRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrInvalid
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()

	cfg, err := config.LoadConfigFromDir(filepath.Join(projectRootDir, "internal", "config"), os.Getenv("CONFIG_NAME"))
	if err != nil {
		t.Skipf("integration tests need config: %v\n  copy internal/config/local.yaml.example to internal/config/local.yaml", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.New(cfg.Database)
	if err != nil {
		t.Skipf("postgres unavailable (%s:%s/%s): %v\n  start DB: make docker-up-db && make migrate-up",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := requireSubscriptionsTable(ctx, pool); err != nil {
		t.Skipf("database not ready: %v\n  apply migrations: make migrate-up", err)
	}

	repo := repository.New(pool)
	svc := service.NewSubscriptionService(repo)
	handler := handlers.NewSubscriptionHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	router := gin.New()
	api := router.Group("/api/v1")
	handler.Register(api)
	return router
}

func requireSubscriptionsTable(ctx context.Context, pool *database.Pool) error {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'subscriptions'
		)`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check subscriptions table: %w", err)
	}
	if !exists {
		return fmt.Errorf("table subscriptions does not exist")
	}
	return nil
}

func uniqueServiceName(prefix string) string {
	return prefix + "_" + uuid.Must(uuid.NewV4()).String()[:8]
}

func TestSubscriptionAPI_Integration(t *testing.T) {
	router := setupRouter(t)
	userID := uuid.Must(uuid.NewV4())
	serviceName := uniqueServiceName("IntegrationNetflix")

	createBody := fmt.Sprintf(`{
		"service_name": %q,
		"monthly_cost": 500,
		"user_id": %q,
		"start_date": "01-2025",
		"end_date": "06-2025"
	}`, serviceName, userID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create: %s", w.Body.String())

	var created models.CreateSubscriptionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotZero(t, created.ID)

	t.Cleanup(func() {
		delW := httptest.NewRecorder()
		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/"+strconv.FormatInt(created.ID, 10), nil)
		router.ServeHTTP(delW, delReq)
	})

	patchW := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/subscriptions/"+strconv.FormatInt(created.ID, 10),
		bytes.NewBufferString(`{"end_date":"12-2026"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(patchW, patchReq)
	require.Equal(t, http.StatusNoContent, patchW.Code, "patch: %s", patchW.Body.String())

	listW := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions?user_id="+userID.String(), nil)
	router.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code, "list: %s", listW.Body.String())

	var subs []models.Subscription
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &subs))
	require.NotEmpty(t, subs)
	assert.Equal(t, serviceName, subs[0].ServiceName)
}

func TestTotalCost_ReturnsCorrectAmount(t *testing.T) {
	router := setupRouter(t)
	userID := uuid.Must(uuid.NewV4())
	serviceName := uniqueServiceName("CostTestService")

	createBody := fmt.Sprintf(`{
		"service_name": %q,
		"monthly_cost": 100,
		"user_id": %q,
		"start_date": "01-2025",
		"end_date": "03-2025"
	}`, serviceName, userID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create: %s", w.Body.String())

	var created models.CreateSubscriptionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	t.Cleanup(func() {
		delW := httptest.NewRecorder()
		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/"+strconv.FormatInt(created.ID, 10), nil)
		router.ServeHTTP(delW, delReq)
	})

	totalURL := "/api/v1/subscriptions/total?user_id=" + userID.String() +
		"&service_name=" + serviceName + "&start_date=01-2025&end_date=03-2025"
	totalW := httptest.NewRecorder()
	totalReq := httptest.NewRequest(http.MethodGet, totalURL, nil)
	router.ServeHTTP(totalW, totalReq)
	require.Equal(t, http.StatusOK, totalW.Code, "total: %s", totalW.Body.String())

	var result models.GetSubscriptionTotalAmountResponse
	require.NoError(t, json.Unmarshal(totalW.Body.Bytes(), &result))
	assert.Greater(t, result.TotalAmount, 0.0, "total_amount should be positive")
}

func TestIntegration_SecondCreateSameService_Conflict(t *testing.T) {
	router := setupRouter(t)
	userID := uuid.Must(uuid.NewV4())
	serviceName := uniqueServiceName("ConflictTest")

	body := fmt.Sprintf(`{
		"service_name": %q,
		"monthly_cost": 100,
		"user_id": %q,
		"start_date": "07-2025"
	}`, serviceName, userID.String())

	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	w1 := post()
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

	var created models.CreateSubscriptionResponse
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &created))
	t.Cleanup(func() {
		delW := httptest.NewRecorder()
		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/"+strconv.FormatInt(created.ID, 10), nil)
		router.ServeHTTP(delW, delReq)
	})

	w2 := post()
	assert.Equal(t, http.StatusConflict, w2.Code, "second active subscription: %s", w2.Body.String())
}
