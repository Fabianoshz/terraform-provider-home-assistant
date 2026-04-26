package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

type MockDevice struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	NameByUser   *string    `json:"name_by_user"`
	Manufacturer *string    `json:"manufacturer"`
	Model        *string    `json:"model"`
	AreaID       *string    `json:"area_id"`
	Connections  [][]string `json:"connections"`
	Identifiers  [][]string `json:"identifiers"`
	DisabledBy   *string    `json:"disabled_by"`
	HWVersion    *string    `json:"hw_version"`
	SerialNumber *string    `json:"serial_number"`
}

type MockEntityRegistry struct {
	EntityID     string  `json:"entity_id"`
	DeviceID     *string `json:"device_id"`
	Name         *string `json:"name"`
	OriginalName *string `json:"original_name"`
	Platform     string  `json:"platform"`
	AreaID       *string `json:"area_id"`
	DisabledBy   *string `json:"disabled_by"`
	Icon         *string `json:"icon"`
}

type MockEntityState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged string         `json:"last_changed"`
	LastUpdated string         `json:"last_updated"`
}

type MockArea struct {
	AreaID  string  `json:"area_id"`
	Name    string  `json:"name"`
	Icon    *string `json:"icon"`
	Picture *string `json:"picture"`
}

type MockAutomation struct {
	ID          string          `json:"id"`
	Alias       string          `json:"alias"`
	Description string          `json:"description"`
	Mode        string          `json:"mode"`
	Trigger     json.RawMessage `json:"triggers"`
	Condition   json.RawMessage `json:"conditions"`
	Action      json.RawMessage `json:"actions"`
}

type MockServerConfig struct {
	Token          string
	Areas          []MockArea
	Automations    []MockAutomation
	Devices        []MockDevice
	EntityRegistry []MockEntityRegistry
	EntityStates   []MockEntityState
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func NewMockHAServer(t *testing.T, cfg MockServerConfig) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/websocket", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("ws upgrade: %v", err)
			return
		}
		defer conn.Close()

		if err := conn.WriteJSON(map[string]string{"type": "auth_required"}); err != nil {
			return
		}

		var authMsg map[string]string
		if err := conn.ReadJSON(&authMsg); err != nil {
			return
		}
		if authMsg["access_token"] != cfg.Token {
			conn.WriteJSON(map[string]any{"type": "auth_invalid", "message": "invalid token"})
			return
		}
		if err := conn.WriteJSON(map[string]string{"type": "auth_ok"}); err != nil {
			return
		}

		for {
			var cmd map[string]any
			if err := conn.ReadJSON(&cmd); err != nil {
				return
			}
			id := int(cmd["id"].(float64))

			var payload any
			switch cmd["type"] {
			case "config/area_registry/list":
				payload = cfg.Areas
			case "config/area_registry/create":
				name, _ := cmd["name"].(string)
				areaID := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
				area := MockArea{AreaID: areaID, Name: name}
				if icon, ok := cmd["icon"].(string); ok {
					area.Icon = Ptr(icon)
				}
				cfg.Areas = append(cfg.Areas, area)
				payload = area
			case "config/area_registry/update":
				areaID, _ := cmd["area_id"].(string)
				for i, a := range cfg.Areas {
					if a.AreaID != areaID {
						continue
					}
					if name, ok := cmd["name"].(string); ok {
						cfg.Areas[i].Name = name
					}
					if v, ok := cmd["icon"]; ok {
						if v == nil {
							cfg.Areas[i].Icon = nil
						} else if s, ok := v.(string); ok {
							cfg.Areas[i].Icon = Ptr(s)
						}
					}
					payload = cfg.Areas[i]
					break
				}
			case "config/area_registry/delete":
				areaID, _ := cmd["area_id"].(string)
				for i, a := range cfg.Areas {
					if a.AreaID == areaID {
						cfg.Areas = append(cfg.Areas[:i], cfg.Areas[i+1:]...)
						break
					}
				}
				payload = map[string]any{}
			case "config/device_registry/list":
				payload = cfg.Devices
			case "config/entity_registry/list":
				payload = cfg.EntityRegistry
			case "config/entity_registry/update":
				entityID, _ := cmd["entity_id"].(string)
				for i, e := range cfg.EntityRegistry {
					if e.EntityID != entityID {
						continue
					}
					if v, ok := cmd["name"]; ok {
						if v == nil {
							cfg.EntityRegistry[i].Name = nil
						} else if s, ok := v.(string); ok {
							cfg.EntityRegistry[i].Name = Ptr(s)
						}
					}
					if v, ok := cmd["icon"]; ok {
						if v == nil {
							cfg.EntityRegistry[i].Icon = nil
						} else if s, ok := v.(string); ok {
							cfg.EntityRegistry[i].Icon = Ptr(s)
						}
					}
					if v, ok := cmd["area_id"]; ok {
						if v == nil {
							cfg.EntityRegistry[i].AreaID = nil
						} else if s, ok := v.(string); ok {
							cfg.EntityRegistry[i].AreaID = Ptr(s)
						}
					}
					if v, ok := cmd["disabled_by"]; ok {
						if v == nil {
							cfg.EntityRegistry[i].DisabledBy = nil
						} else if s, ok := v.(string); ok {
							cfg.EntityRegistry[i].DisabledBy = Ptr(s)
						}
					}
					// HA wraps the result in entity_entry for this command
					result, _ := json.Marshal(map[string]any{"entity_entry": cfg.EntityRegistry[i]})
					conn.WriteJSON(map[string]any{
						"id": id, "type": "result", "success": true,
						"result": json.RawMessage(result),
					})
					payload = nil // already sent
					break
				}
				if payload == nil {
					continue
				}
			case "config/device_registry/update":
				deviceID, _ := cmd["device_id"].(string)
				for i, d := range cfg.Devices {
					if d.ID != deviceID {
						continue
					}
					if v, ok := cmd["name_by_user"]; ok {
						if v == nil {
							cfg.Devices[i].NameByUser = nil
						} else if s, ok := v.(string); ok {
							cfg.Devices[i].NameByUser = Ptr(s)
						}
					}
					if v, ok := cmd["area_id"]; ok {
						if v == nil {
							cfg.Devices[i].AreaID = nil
						} else if s, ok := v.(string); ok {
							cfg.Devices[i].AreaID = Ptr(s)
						}
					}
					if v, ok := cmd["disabled_by"]; ok {
						if v == nil {
							cfg.Devices[i].DisabledBy = nil
						} else if s, ok := v.(string); ok {
							cfg.Devices[i].DisabledBy = Ptr(s)
						}
					}
					payload = cfg.Devices[i]
					break
				}
			default:
				continue
			}

			result, _ := json.Marshal(payload)
			conn.WriteJSON(map[string]any{
				"id":      id,
				"type":    "result",
				"success": true,
				"result":  json.RawMessage(result),
			})
		}
	})

	mux.HandleFunc("/api/config/automation/config/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+cfg.Token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/config/automation/config/")
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			for _, a := range cfg.Automations {
				if a.ID == id {
					json.NewEncoder(w).Encode(a)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPost:
			var a MockAutomation
			json.NewDecoder(r.Body).Decode(&a)
			a.ID = id
			for i, existing := range cfg.Automations {
				if existing.ID == id {
					cfg.Automations[i] = a
					json.NewEncoder(w).Encode(a)
					return
				}
			}
			cfg.Automations = append(cfg.Automations, a)
			json.NewEncoder(w).Encode(a)
		case http.MethodDelete:
			for i, a := range cfg.Automations {
				if a.ID == id {
					cfg.Automations = append(cfg.Automations[:i], cfg.Automations[i+1:]...)
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	})

	mux.HandleFunc("/api/config/automation/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+cfg.Token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(cfg.Automations)
		case http.MethodPost:
			var a MockAutomation
			json.NewDecoder(r.Body).Decode(&a)
			a.ID = fmt.Sprintf("automation_%d", len(cfg.Automations)+1)
			cfg.Automations = append(cfg.Automations, a)
			json.NewEncoder(w).Encode(a)
		}
	})

	mux.HandleFunc("/api/states", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+cfg.Token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg.EntityStates)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func Ptr(s string) *string { return &s }
