package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/api/mocks"
	"pulsewatch-backend/internal/models"
)

func TestAlertSettingsHandler_Get(t *testing.T) {
	t.Run("success includes demo_mode flag", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		repo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{EmailEnabled: true}, nil)
		h := NewAlertSettingsHandler(repo, true)

		w := httptest.NewRecorder()
		h.Get(w, httptest.NewRequest(http.MethodGet, "/api/settings/alerts", nil))

		require.Equal(t, http.StatusOK, w.Code)
		var got alertSettingsResponse
		decodeBody(t, w, &got)
		assert.True(t, got.EmailEnabled)
		assert.True(t, got.DemoMode)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		repo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, errors.New("db down"))
		h := NewAlertSettingsHandler(repo, false)

		w := httptest.NewRecorder()
		h.Get(w, httptest.NewRequest(http.MethodGet, "/api/settings/alerts", nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAlertSettingsHandler_Update(t *testing.T) {
	t.Run("invalid JSON body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		h := NewAlertSettingsHandler(repo, false)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings/alerts", strings.NewReader("{bad"))
		h.Update(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("email enabled requires an address", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		h := NewAlertSettingsHandler(repo, false)

		in := models.UpdateAlertSettingsInput{EmailEnabled: true, EmailAddress: ""}
		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "email_address is required")
	})

	t.Run("email enabled requires a valid address", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		h := NewAlertSettingsHandler(repo, false)

		in := models.UpdateAlertSettingsInput{EmailEnabled: true, EmailAddress: "not-an-email"}
		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "not a valid email address")
	})

	t.Run("webhook enabled requires a url", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		h := NewAlertSettingsHandler(repo, false)

		in := models.UpdateAlertSettingsInput{WebhookEnabled: true, WebhookURL: ""}
		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "webhook_url is required")
	})

	t.Run("webhook enabled requires a valid non-SSRF url", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		h := NewAlertSettingsHandler(repo, false)

		in := models.UpdateAlertSettingsInput{WebhookEnabled: true, WebhookURL: "http://127.0.0.1/"}
		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, in)))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(models.AlertSettings{}, errors.New("db down"))
		h := NewAlertSettingsHandler(repo, false)

		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, models.UpdateAlertSettingsInput{})))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with both channels disabled skips validation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(models.AlertSettings{}, nil)
		h := NewAlertSettingsHandler(repo, false)

		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, models.UpdateAlertSettingsInput{})))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with both channels enabled and valid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAlertSettingsRepository(ctrl)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(models.AlertSettings{EmailEnabled: true, WebhookEnabled: true}, nil)
		h := NewAlertSettingsHandler(repo, false)

		in := models.UpdateAlertSettingsInput{
			EmailEnabled: true, EmailAddress: "me@example.com",
			WebhookEnabled: true, WebhookURL: "https://example.com/hooks/pulsewatch",
		}
		w := httptest.NewRecorder()
		h.Update(w, httptest.NewRequest(http.MethodPut, "/api/settings/alerts", jsonBody(t, in)))

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
