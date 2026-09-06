package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/api/mocks"
	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/store"
)

const validID = "550e8400-e29b-41d4-a716-446655440000"

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(w.Body).Decode(v))
}

// --- List ---

func TestMonitorHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{{ID: validID, Name: "a"}}, nil)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.List(w, httptest.NewRequest(http.MethodGet, "/api/monitors", nil))

		require.Equal(t, http.StatusOK, w.Code)
		var got []models.Monitor
		decodeBody(t, w, &got)
		assert.Len(t, got, 1)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().List(gomock.Any()).Return(nil, errors.New("db down"))

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.List(w, httptest.NewRequest(http.MethodGet, "/api/monitors", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- Create ---

func validCreateInput() models.CreateMonitorInput {
	return models.CreateMonitorInput{
		Name:               "Production API",
		URL:                "https://example.com/health",
		Method:             "GET",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	}
}

func TestMonitorHandler_Create(t *testing.T) {
	t.Run("demo cap: count check fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Count(gomock.Any()).Return(0, errors.New("db down"))

		h := NewMonitorHandler(repo, 5)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, validCreateInput()))
		h.Create(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to check monitor count")
	})

	t.Run("demo cap reached", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Count(gomock.Any()).Return(5, nil)

		h := NewMonitorHandler(repo, 5)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, validCreateInput()))
		h.Create(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Demo limit reached")
	})

	t.Run("demo cap disabled skips count check entirely", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		// No .EXPECT().Count(...) at all: maxMonitors=0 must never call Count.
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(models.Monitor{ID: validID}, nil)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, validCreateInput()))
		h.Create(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader("{not json"))
		h.Create(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request body")
	})

	t.Run("name required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.Name = ""

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "name is required")
	})

	t.Run("name too long", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.Name = strings.Repeat("a", maxNameLength+1)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("url required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.URL = ""

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "url is required")
	})

	t.Run("url fails SSRF validation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.URL = "http://127.0.0.1:9999/"

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "local or internal address")
	})

	t.Run("method defaults to GET when empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.Method = ""
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, got models.CreateMonitorInput) (models.Monitor, error) {
				assert.Equal(t, "GET", got.Method)
				return models.Monitor{ID: validID}, nil
			},
		)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid explicit method", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.Method = "DELETE"

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("interval defaults to 60 when zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.IntervalSeconds = 0
		in.TimeoutSeconds = 0
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, got models.CreateMonitorInput) (models.Monitor, error) {
				assert.Equal(t, 60, got.IntervalSeconds)
				assert.Equal(t, 10, got.TimeoutSeconds)
				return models.Monitor{ID: validID}, nil
			},
		)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("explicit timeout greater than interval is rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.IntervalSeconds = 5
		in.TimeoutSeconds = 30

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "timeout_seconds cannot be greater than interval_seconds")
	})

	t.Run("default timeout longer than a short custom interval is clamped, not rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.IntervalSeconds = 5
		in.TimeoutSeconds = 0 // not provided -> defaults to 10, which exceeds interval=5
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, got models.CreateMonitorInput) (models.Monitor, error) {
				assert.Equal(t, 5, got.TimeoutSeconds)
				return models.Monitor{ID: validID}, nil
			},
		)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("expected status code defaults to 200 when zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.ExpectedStatusCode = 0
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, got models.CreateMonitorInput) (models.Monitor, error) {
				assert.Equal(t, 200, got.ExpectedStatusCode)
				return models.Monitor{ID: validID}, nil
			},
		)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid explicit expected status code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		in := validCreateInput()
		in.ExpectedStatusCode = 50

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("repo create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(models.Monitor{}, errors.New("db down"))

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, validCreateInput())))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(models.Monitor{ID: validID, Name: "Production API"}, nil)

		h := NewMonitorHandler(repo, 0)
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest(http.MethodPost, "/api/monitors", jsonBody(t, validCreateInput())))

		require.Equal(t, http.StatusCreated, w.Code)
		var got models.Monitor
		decodeBody(t, w, &got)
		assert.Equal(t, validID, got.ID)
	})
}

// --- Get ---

func monitorRouter(t *testing.T, h *MonitorHandler) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/api/monitors/{id}", h.Get)
	r.Put("/api/monitors/{id}", h.Update)
	r.Delete("/api/monitors/{id}", h.Delete)
	return r
}

func TestMonitorHandler_Get(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/not-a-uuid", nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{}, store.ErrMonitorNotFound)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{}, errors.New("db down"))
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID}, nil)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- Update ---

func TestMonitorHandler_Update(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(n int) *int { return &n }

	t.Run("invalid id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/monitors/not-a-uuid", jsonBody(t, models.UpdateMonitorInput{}))
		monitorRouter(t, h).ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, strings.NewReader("{bad"))
		monitorRouter(t, h).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr("")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "name cannot be empty")
	})

	t.Run("name too long rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr(strings.Repeat("a", maxNameLength+1))}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty url rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{URL: strPtr("")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "url cannot be empty")
	})

	t.Run("SSRF-invalid url rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{URL: strPtr("http://169.254.169.254/")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid method rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Method: strPtr("DELETE")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid expected status code rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{ExpectedStatusCode: intPtr(999)}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("non-positive interval rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{IntervalSeconds: intPtr(0)}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "interval_seconds must be positive")
	})

	t.Run("non-positive timeout rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{TimeoutSeconds: intPtr(-1)}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "timeout_seconds must be positive")
	})

	t.Run("current monitor not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{}, store.ErrMonitorNotFound)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr("New name")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("current monitor lookup error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{}, errors.New("db down"))
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr("New name")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("explicit timeout greater than effective interval is rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID, IntervalSeconds: 30, TimeoutSeconds: 10}, nil)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{TimeoutSeconds: intPtr(60)}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "timeout_seconds cannot be greater than interval_seconds")
	})

	t.Run("shrinking interval below the untouched timeout clamps it instead of rejecting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID, IntervalSeconds: 60, TimeoutSeconds: 30}, nil)
		repo.EXPECT().Update(gomock.Any(), validID, gomock.Any()).DoAndReturn(
			func(_ any, _ string, got models.UpdateMonitorInput) (models.Monitor, error) {
				require.NotNil(t, got.TimeoutSeconds)
				assert.Equal(t, 10, *got.TimeoutSeconds)
				return models.Monitor{ID: validID}, nil
			},
		)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{IntervalSeconds: intPtr(10)}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("update not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID, IntervalSeconds: 60, TimeoutSeconds: 10}, nil)
		repo.EXPECT().Update(gomock.Any(), validID, gomock.Any()).Return(models.Monitor{}, store.ErrMonitorNotFound)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr("New name")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID, IntervalSeconds: 60, TimeoutSeconds: 10}, nil)
		repo.EXPECT().Update(gomock.Any(), validID, gomock.Any()).Return(models.Monitor{}, errors.New("db down"))
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Name: strPtr("New name")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with valid method normalization", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), validID).Return(models.Monitor{ID: validID, IntervalSeconds: 60, TimeoutSeconds: 10}, nil)
		repo.EXPECT().Update(gomock.Any(), validID, gomock.Any()).DoAndReturn(
			func(_ any, _ string, got models.UpdateMonitorInput) (models.Monitor, error) {
				require.NotNil(t, got.Method)
				assert.Equal(t, "POST", *got.Method)
				return models.Monitor{ID: validID}, nil
			},
		)
		h := NewMonitorHandler(repo, 0)

		in := models.UpdateMonitorInput{Method: strPtr("post")}
		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/monitors/"+validID, jsonBody(t, in)))

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- Delete ---

func TestMonitorHandler_Delete(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/monitors/not-a-uuid", nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Delete(gomock.Any(), validID).Return(store.ErrMonitorNotFound)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Delete(gomock.Any(), validID).Return(errors.New("db down"))
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockMonitorRepository(ctrl)
		repo.EXPECT().Delete(gomock.Any(), validID).Return(nil)
		h := NewMonitorHandler(repo, 0)

		w := httptest.NewRecorder()
		monitorRouter(t, h).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/monitors/"+validID, nil))

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
