package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/api/mocks"
	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/websocket"
)

// buildTestRouter wires NewRouter with mock repos so this test exercises
// only routing (method/path -> handler dispatch, healthz, CORS), not
// handler business logic already covered by the handler-level tests.
func buildTestRouter(t *testing.T) http.Handler {
	t.Helper()
	ctrl := gomock.NewController(t)

	monitorRepo := mocks.NewMockMonitorRepository(ctrl)
	monitorRepo.EXPECT().List(gomock.Any()).Return([]models.Monitor{}, nil).AnyTimes()
	monitorRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(models.Monitor{}, nil).AnyTimes()

	checkResultRepo := mocks.NewMockCheckResultRepository(ctrl)
	checkResultRepo.EXPECT().History(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	checkResultRepo.EXPECT().UptimePercentage(gomock.Any(), gomock.Any(), gomock.Any()).Return(0.0, nil).AnyTimes()

	settingsRepo := mocks.NewMockAlertSettingsRepository(ctrl)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, nil).AnyTimes()

	incidentRepo := mocks.NewMockIncidentRepository(ctrl)
	incidentRepo.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, 0, nil).AnyTimes()

	monitorHandler := NewMonitorHandler(monitorRepo, 0)
	historyHandler := NewHistoryHandler(checkResultRepo)
	alertSettingsHandler := NewAlertSettingsHandler(settingsRepo, false)
	incidentHandler := NewIncidentHandler(incidentRepo)
	hub := websocket.NewHub()

	return NewRouter(monitorHandler, historyHandler, alertSettingsHandler, incidentHandler, hub, func() [][]byte { return nil })
}

func TestRouter_Healthz(t *testing.T) {
	router := buildTestRouter(t)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestRouter_CORSPreflight(t *testing.T) {
	router := buildTestRouter(t)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/api/monitors", nil))

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRouter_Dispatch(t *testing.T) {
	router := buildTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"list monitors", http.MethodGet, "/api/monitors/"},
		{"create monitor invalid body still routes", http.MethodPost, "/api/monitors/"},
		{"get monitor", http.MethodGet, "/api/monitors/" + validID},
		{"monitor history", http.MethodGet, "/api/monitors/" + validID + "/history"},
		{"monitor incidents", http.MethodGet, "/api/monitors/" + validID + "/incidents"},
		{"get alert settings", http.MethodGet, "/api/settings/alerts"},
		{"global incidents", http.MethodGet, "/api/incidents"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			// Routing succeeded as long as chi didn't produce its own 404/405;
			// handler-level correctness is covered elsewhere.
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route should have matched")
			assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code, "method should be allowed on this route")
		})
	}
}

// The /ws route itself is exercised here just to cover its registration
// line; a real WebSocket handshake needs a hijackable connection, which
// httptest.NewRecorder doesn't provide, so the actual upgrade/broadcast
// behavior is covered in internal/websocket's own tests against a real
// httptest.NewServer.
func TestRouter_WebSocketRouteRegistered(t *testing.T) {
	router := buildTestRouter(t)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws", nil))

	assert.NotEqual(t, http.StatusNotFound, w.Code, "route should have matched")
}

func TestRouter_UnknownRouteIs404(t *testing.T) {
	router := buildTestRouter(t)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
