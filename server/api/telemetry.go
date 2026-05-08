// Telemetry endpoint and adapters.
//
// The deprecated webapp called `trackEvent('insights', '<event>',
// properties?)` from `actions/telemetry_actions`, which the host's
// RudderStack integration forwarded to the analytics backend. Plugins
// don't have access to that pipeline, so this plugin's telemetry endpoint
// records events to the Mattermost server log (LogInfo) instead. Admins
// can scrape the structured log records and feed them into whatever
// analytics tool they already run.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// noopTelemetry drops all events. Used by tests by default.
type noopTelemetry struct{}

func (noopTelemetry) TrackInsightsEvent(string, map[string]any) {}

// pluginAPITelemetry forwards events to the host's structured logger.
type pluginAPITelemetry struct {
	client *pluginapi.Client
}

func (t pluginAPITelemetry) TrackInsightsEvent(event string, properties map[string]any) {
	args := []any{"category", "insights", "event", event}
	for k, v := range properties {
		args = append(args, k, v)
	}
	t.client.Log.Info("insights telemetry", args...)
}

// telemetryRequest matches the deprecated `trackEvent(category, event,
// properties?)` shape — category is fixed at "insights" so the wire
// format only carries event + properties.
type telemetryRequest struct {
	Event      string         `json:"event"`
	Properties map[string]any `json:"properties,omitempty"`
}

// handleTelemetry handles
// POST /plugins/insights/api/v1/telemetry
//
// Auth: any authenticated user. Guests are allowed to record events too
// (the deprecated `trackEvent` had no guest gate).
func (a *API) handleTelemetry(w http.ResponseWriter, r *http.Request, userID string) {
	var req telemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid telemetry payload: "+err.Error())
		return
	}
	if req.Event == "" {
		writeJSONError(w, http.StatusBadRequest, "event is required")
		return
	}

	props := req.Properties
	if props == nil {
		props = map[string]any{}
	}
	props["user_id"] = userID
	a.telemetry.TrackInsightsEvent(req.Event, props)

	w.WriteHeader(http.StatusNoContent)
}
