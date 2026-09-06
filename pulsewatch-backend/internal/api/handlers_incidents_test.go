package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/api/mocks"
	"pulsewatch-backend/internal/models"
)

func incidentRouter(h *IncidentHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/incidents", h.List)
	r.Get("/api/monitors/{id}/incidents", h.ListForMonitor)
	return r
}

func TestIncidentHandler_List(t *testing.T) {
	t.Run("default paging", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", defaultIncidentLimit, 0).
			Return([]models.Incident{{ID: "i1"}}, 1, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents", nil))

		require.Equal(t, http.StatusOK, w.Code)
		var got incidentPage
		decodeBody(t, w, &got)
		assert.Equal(t, 1, got.Total)
		assert.Len(t, got.Incidents, 1)
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", 25, 50).Return(nil, 0, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents?limit=25&offset=50", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("limit clamped to max", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", maxIncidentLimit, 0).Return(nil, 0, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents?limit=99999", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid limit falls back to default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", defaultIncidentLimit, 0).Return(nil, 0, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents?limit=not-a-number&offset=-5", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative limit falls back to default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", defaultIncidentLimit, 0).Return(nil, 0, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents?limit=-1", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), "", defaultIncidentLimit, 0).Return(nil, 0, errors.New("db down"))
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/incidents", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestIncidentHandler_ListForMonitor(t *testing.T) {
	t.Run("invalid monitor id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/not-a-uuid/incidents", nil))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), validID, defaultIncidentLimit, 0).
			Return([]models.Incident{{ID: "i1", MonitorID: validID}}, 1, nil)
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/incidents", nil))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockIncidentRepository(ctrl)
		repo.EXPECT().List(gomock.Any(), validID, defaultIncidentLimit, 0).Return(nil, 0, errors.New("db down"))
		h := NewIncidentHandler(repo)

		w := httptest.NewRecorder()
		incidentRouter(h).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/monitors/"+validID+"/incidents", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
