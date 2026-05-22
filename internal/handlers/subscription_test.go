package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LimeAsnet/SubAggregator/internal/handlers/mocks"
	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testUserID = uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func newTestHandler(svc *mocks.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func setupRouter(h *SubscriptionHandler) *gin.Engine {
	r := gin.New()
	h.Register(r)
	return r
}

func TestHandler_Create_Success(t *testing.T) {
	svc := &mocks.SubscriptionService{
		CreateFn: func(_ context.Context, req models.CreateSubscriptionRequest) (int64, error) {
			assert.Equal(t, "Netflix", req.ServiceName)
			return 1, nil
		},
	}

	body := `{"service_name":"Netflix","monthly_cost":599,"user_id":"550e8400-e29b-41d4-a716-446655440000","start_date":"07-2025"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_ServiceInvalidDate(t *testing.T) {
	svc := &mocks.SubscriptionService{
		CreateFn: func(context.Context, models.CreateSubscriptionRequest) (int64, error) {
			return 0, service.ErrInvalidDate
		},
	}

	body := `{"service_name":"Netflix","monthly_cost":599,"user_id":"550e8400-e29b-41d4-a716-446655440000","start_date":"07-2025"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_InvalidUserID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/subscriptions?user_id=bad", nil)
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_MissingUserID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TotalCost_NotFound(t *testing.T) {
	svc := &mocks.SubscriptionService{
		TotalCostFn: func(context.Context, uuid.UUID, string, string, string) (models.GetSubscriptionTotalAmountResponse, error) {
			return models.GetSubscriptionTotalAmountResponse{}, service.ErrNotFound
		},
	}

	url := "/subscriptions/total?user_id=" + testUserID.String() + "&service_name=Netflix&start_date=01-2025&end_date=12-2025"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_TotalCost_Success(t *testing.T) {
	svc := &mocks.SubscriptionService{
		TotalCostFn: func(_ context.Context, userID uuid.UUID, serviceName, startDate, endDate string) (models.GetSubscriptionTotalAmountResponse, error) {
			assert.Equal(t, testUserID, userID)
			assert.Equal(t, "Netflix", serviceName)
			return models.GetSubscriptionTotalAmountResponse{TotalAmount: 1234.56}, nil
		},
	}

	url := "/subscriptions/total?user_id=" + testUserID.String() + "&service_name=Netflix&start_date=01-2025&end_date=12-2025"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result models.GetSubscriptionTotalAmountResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, 1234.56, result.TotalAmount)
}

func TestHandler_Update_MissingEndDate(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/subscriptions/1", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/subscriptions/abc", bytes.NewBufferString(`{"end_date":"12-2026"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_Success(t *testing.T) {
	svc := &mocks.SubscriptionService{
		UpdateFn: func(_ context.Context, id int64, req models.PatchSubscriptionRequest) error {
			assert.Equal(t, int64(1), id)
			assert.Equal(t, "12-2026", req.EndDate)
			return nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/subscriptions/1", bytes.NewBufferString(`{"end_date":"12-2026"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Delete_Success(t *testing.T) {
	var deleted int64
	svc := &mocks.SubscriptionService{
		DeleteFn: func(_ context.Context, id int64) error {
			deleted = id
			return nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/subscriptions/5", nil)
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, int64(5), deleted)
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/subscriptions/xyz", nil)
	setupRouter(newTestHandler(&mocks.SubscriptionService{})).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_Success(t *testing.T) {
	svc := &mocks.SubscriptionService{
		ListFn: func(_ context.Context, id uuid.UUID) ([]models.Subscription, error) {
			return []models.Subscription{{ID: 1, ServiceName: "Netflix", UserID: id}}, nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/subscriptions?user_id="+testUserID.String(), nil)
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var subs []models.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &subs))
	require.Len(t, subs, 1)
}

func TestHandler_Create_ActiveSubscription(t *testing.T) {
	svc := &mocks.SubscriptionService{
		CreateFn: func(context.Context, models.CreateSubscriptionRequest) (int64, error) {
			return 0, service.ErrActiveSubscription
		},
	}

	body := `{"service_name":"Netflix","monthly_cost":599,"user_id":"550e8400-e29b-41d4-a716-446655440000","start_date":"07-2025"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, service.ErrActiveSubscription.Error(), resp["error"])
}

func TestHandler_Update_NotFound(t *testing.T) {
	svc := &mocks.SubscriptionService{
		UpdateFn: func(context.Context, int64, models.PatchSubscriptionRequest) error {
			return service.ErrNotFound
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/subscriptions/99", bytes.NewBufferString(`{"end_date":"12-2026"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, service.ErrNotFound.Error(), resp["error"])
}

func TestHandler_Update_EndBeforeStart(t *testing.T) {
	svc := &mocks.SubscriptionService{
		UpdateFn: func(context.Context, int64, models.PatchSubscriptionRequest) error {
			return service.ErrEndBeforeStart
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/subscriptions/1", bytes.NewBufferString(`{"end_date":"01-2020"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, service.ErrEndBeforeStart.Error(), resp["error"])
}

func TestHandler_Create_RepositoryErrorViaService(t *testing.T) {
	svc := &mocks.SubscriptionService{
		CreateFn: func(context.Context, models.CreateSubscriptionRequest) (int64, error) {
			return 0, errors.New("db error")
		},
	}

	body := `{"service_name":"Netflix","monthly_cost":599,"user_id":"550e8400-e29b-41d4-a716-446655440000","start_date":"07-2025"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(newTestHandler(svc)).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "db error", resp["error"])
}
