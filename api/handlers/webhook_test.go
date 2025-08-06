package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"webhook-processor/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(event models.WebhookEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func TestHandleWebhook(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name       string
		clientID   string
		payload    interface{}
		setupMock  func(*MockPublisher)
		wantStatus int
	}{
		{
			name:     "Valid request",
			clientID: "test-client",
			payload: models.WebhookEvent{
				Event:        "Campaign Sent",
				CampaignName: "Test Campaign",
				CampaignID:   "123",
			},
			setupMock: func(m *MockPublisher) {
				m.On("Publish", mock.Anything).Return(nil)
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:     "Missing client ID",
			clientID: "",
			payload: models.WebhookEvent{
				Event: "Campaign Sent",
			},
			setupMock:  func(m *MockPublisher) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid payload",
			clientID:   "test-client",
			payload:    "invalid",
			setupMock:  func(m *MockPublisher) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock publisher
			mockPub := new(MockPublisher)
			tt.setupMock(mockPub)

			// Create handler
			handler := NewWebhookHandler(logger, mockPub)

			// Create request
			payload, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			if tt.clientID != "" {
				req.Header.Set("X-Client-ID", tt.clientID)
			}

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.HandleWebhook(c)

			// Assert response
			assert.Equal(t, tt.wantStatus, w.Code)
			mockPub.AssertExpectations(t)
		})
	}
}
