package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
)

type capturingTelemetry struct {
	events []capturedEvent
}

type capturedEvent struct {
	event string
	props map[string]any
}

func (c *capturingTelemetry) TrackInsightsEvent(event string, props map[string]any) {
	c.events = append(c.events, capturedEvent{event: event, props: props})
}

func newTelemetryAPI() (*API, *capturingTelemetry) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{"u1": {Id: "u1"}}}
	cap := &capturingTelemetry{}
	return NewWithTelemetry(auth, nil, &apitest.DirectoryStub{}, &apitest.StoreStub{}, cap), cap
}

func TestTelemetry_recordsEventForAuthenticatedUser(t *testing.T) {
	api, cap := newTelemetryAPI()

	body := bytes.NewBufferString(`{"event":"sidebar_open_insights","properties":{"foo":"bar"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", body)
	req.Header.Set("Mattermost-User-Id", "u1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(cap.events) != 1 {
		t.Fatalf("expected one event, got %v", cap.events)
	}
	if cap.events[0].event != "sidebar_open_insights" {
		t.Errorf("event mismatch: got %s", cap.events[0].event)
	}
	if cap.events[0].props["foo"] != "bar" {
		t.Errorf("missing 'foo' property: got %v", cap.events[0].props)
	}
	// user_id is automatically attached.
	if cap.events[0].props["user_id"] != "u1" {
		t.Errorf("missing user_id property: got %v", cap.events[0].props)
	}
}

func TestTelemetry_rejectsMissingEvent(t *testing.T) {
	api, _ := newTelemetryAPI()

	body := bytes.NewBufferString(`{"properties":{"foo":"bar"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", body)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTelemetry_rejectsUnauthenticated(t *testing.T) {
	api, _ := newTelemetryAPI()

	body := bytes.NewBufferString(`{"event":"foo"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", body)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
