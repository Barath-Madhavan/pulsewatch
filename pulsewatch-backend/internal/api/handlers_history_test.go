package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/api/mocks"
	"pulsewatch-backend/internal/store"
)

func historyRouter(h *HistoryHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/monitors/{id}/history", h.Get)
	return r
}

func TestHistoryHandler_Get(t *testing.T) {
	t.Run("invalid monitor id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/not-a-uuid/history", nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("defaults to 24h range", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		repo.EXPECT().History(gomock.Any(), validID, gomock.Any(), 5*time.Minute).
			Return([]store.HistoryBucket{{TotalChecks: 1}}, nil)
		repo.EXPECT().UptimePercentage(gomock.Any(), validID, gomock.Any()).Return(99.5, nil)
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history", nil))

		require.Equal(t, http.StatusOK, w.Code)
		var got historyResponse
		decodeBody(t, w, &got)
		assert.Equal(t, "24h", got.Range)
		assert.Equal(t, 99.5, got.UptimePercentage)
	})

	t.Run("7d range uses hourly buckets", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		repo.EXPECT().History(gomock.Any(), validID, gomock.Any(), time.Hour).Return(nil, nil)
		repo.EXPECT().UptimePercentage(gomock.Any(), validID, gomock.Any()).Return(100.0, nil)
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history?range=7d", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("30d range uses 6h buckets", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		repo.EXPECT().History(gomock.Any(), validID, gomock.Any(), 6*time.Hour).Return(nil, nil)
		repo.EXPECT().UptimePercentage(gomock.Any(), validID, gomock.Any()).Return(100.0, nil)
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history?range=30d", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid range rejected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history?range=1y", nil))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "range must be one of 24h, 7d, 30d")
	})

	t.Run("history repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		repo.EXPECT().History(gomock.Any(), validID, gomock.Any(), gomock.Any()).Return(nil, errors.New("db down"))
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("uptime percentage repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCheckResultRepository(ctrl)
		repo.EXPECT().History(gomock.Any(), validID, gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().UptimePercentage(gomock.Any(), validID, gomock.Any()).Return(0.0, errors.New("db down"))
		h := NewHistoryHandler(repo)

		w := httptest.NewRecorder()
		historyRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/history", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
