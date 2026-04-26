package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	baseURL string
	token   string
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

func (c *Client) Verify(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (c *Client) wsURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/api/websocket"
	return u.String(), nil
}

func (c *Client) dial(ctx context.Context) (*websocket.Conn, error) {
	wsURL, err := c.wsURL()
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("connecting to Home Assistant WebSocket: %w", err)
	}

	// Expect auth_required
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("reading auth_required: %w", err)
	}
	if msg["type"] != "auth_required" {
		conn.Close()
		return nil, fmt.Errorf("expected auth_required, got %v", msg["type"])
	}

	// Send auth
	if err := conn.WriteJSON(map[string]string{"type": "auth", "access_token": c.token}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sending auth: %w", err)
	}

	// Expect auth_ok
	if err := conn.ReadJSON(&msg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("reading auth response: %w", err)
	}
	if msg["type"] == "auth_invalid" {
		conn.Close()
		return nil, fmt.Errorf("invalid token: %v", msg["message"])
	}
	if msg["type"] != "auth_ok" {
		conn.Close()
		return nil, fmt.Errorf("unexpected auth response: %v", msg["type"])
	}

	return conn, nil
}

type Device struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	NameByUser       *string  `json:"name_by_user"`
	Manufacturer     *string  `json:"manufacturer"`
	Model            *string  `json:"model"`
	AreaID           *string  `json:"area_id"`
	ConfigEntries    []string `json:"config_entries"`
	Connections      [][]string `json:"connections"`
	Identifiers      [][]string `json:"identifiers"`
	DisabledBy       *string  `json:"disabled_by"`
	EntryType        *string  `json:"entry_type"`
	HWVersion        *string  `json:"hw_version"`
	SerialNumber     *string  `json:"serial_number"`
	ViaDeviceID      *string  `json:"via_device_id"`
}

type wsResult struct {
	ID      int             `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Error   *wsError        `json:"error"`
}

type wsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EntityState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged string         `json:"last_changed"`
	LastUpdated string         `json:"last_updated"`
}

var restClient = &http.Client{Timeout: 30 * time.Second}

func (c *Client) restDo(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}
	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return restClient.Do(req)
}

type Automation struct {
	ID          string          `json:"id"`
	Alias       string          `json:"alias"`
	Description string          `json:"description"`
	Mode        string          `json:"mode"`
	Trigger     json.RawMessage `json:"trigger"`
	Condition   json.RawMessage `json:"condition"`
	Action      json.RawMessage `json:"action"`
}

func (c *Client) GetAutomations(ctx context.Context) ([]Automation, error) {
	resp, err := c.restDo(ctx, http.MethodGet, "/api/config/automation/config", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching automations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching automations", resp.StatusCode)
	}

	var automations []Automation
	if err := json.NewDecoder(resp.Body).Decode(&automations); err != nil {
		return nil, fmt.Errorf("decoding automations: %w", err)
	}
	return automations, nil
}

func (c *Client) GetAutomation(ctx context.Context, id string) (*Automation, error) {
	resp, err := c.restDo(ctx, http.MethodGet, "/api/config/automation/config/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching automation", resp.StatusCode)
	}

	var automation Automation
	if err := json.NewDecoder(resp.Body).Decode(&automation); err != nil {
		return nil, fmt.Errorf("decoding automation: %w", err)
	}
	return &automation, nil
}

func (c *Client) CreateAutomation(ctx context.Context, a Automation) (*Automation, error) {
	body, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("encoding automation: %w", err)
	}

	resp, err := c.restDo(ctx, http.MethodPost, "/api/config/automation/config", body)
	if err != nil {
		return nil, fmt.Errorf("creating automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d creating automation", resp.StatusCode)
	}

	var created Automation
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decoding created automation: %w", err)
	}
	return &created, nil
}

func (c *Client) UpdateAutomation(ctx context.Context, a Automation) (*Automation, error) {
	body, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("encoding automation: %w", err)
	}

	resp, err := c.restDo(ctx, http.MethodPost, "/api/config/automation/config/"+a.ID, body)
	if err != nil {
		return nil, fmt.Errorf("updating automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d updating automation", resp.StatusCode)
	}

	var updated Automation
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return nil, fmt.Errorf("decoding updated automation: %w", err)
	}
	return &updated, nil
}

func (c *Client) DeleteAutomation(ctx context.Context, id string) error {
	resp, err := c.restDo(ctx, http.MethodDelete, "/api/config/automation/config/"+id, nil)
	if err != nil {
		return fmt.Errorf("deleting automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d deleting automation", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetStates(ctx context.Context) ([]EntityState, error) {
	resp, err := c.restDo(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching states: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching states", resp.StatusCode)
	}

	var states []EntityState
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("decoding states: %w", err)
	}
	return states, nil
}

type Area struct {
	AreaID  string   `json:"area_id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Icon    *string  `json:"icon"`
	Picture *string  `json:"picture"`
}

func (c *Client) GetAreas(ctx context.Context) ([]Area, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{"id": 1, "type": "config/area_registry/list"}); err != nil {
		return nil, fmt.Errorf("sending area_registry/list: %w", err)
	}

	var result wsResult
	if err := conn.ReadJSON(&result); err != nil {
		return nil, fmt.Errorf("reading area registry response: %w", err)
	}
	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf("area registry error %s: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("area registry request failed")
	}

	var areas []Area
	if err := json.Unmarshal(result.Result, &areas); err != nil {
		return nil, fmt.Errorf("decoding areas: %w", err)
	}
	return areas, nil
}

func (c *Client) GetArea(ctx context.Context, areaID string) (*Area, error) {
	areas, err := c.GetAreas(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range areas {
		if a.AreaID == areaID {
			return &a, nil
		}
	}
	return nil, nil
}

func (c *Client) CreateArea(ctx context.Context, name string, icon *string) (*Area, error) {
	fields := map[string]any{"name": name}
	if icon != nil {
		fields["icon"] = *icon
	}
	return c.areaCommand(ctx, "config/area_registry/create", fields)
}

func (c *Client) UpdateArea(ctx context.Context, areaID, name string, icon *string) (*Area, error) {
	fields := map[string]any{
		"area_id": areaID,
		"name":    name,
	}
	if icon != nil {
		fields["icon"] = *icon
	}
	return c.areaCommand(ctx, "config/area_registry/update", fields)
}

func (c *Client) DeleteArea(ctx context.Context, areaID string) error {
	_, err := c.areaCommand(ctx, "config/area_registry/delete", map[string]any{
		"area_id": areaID,
	})
	return err
}

func (c *Client) areaCommand(ctx context.Context, cmdType string, fields map[string]any) (*Area, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payload := map[string]any{"id": 1, "type": cmdType}
	for k, v := range fields {
		payload[k] = v
	}

	if err := conn.WriteJSON(payload); err != nil {
		return nil, fmt.Errorf("sending %s: %w", cmdType, err)
	}

	var result wsResult
	if err := conn.ReadJSON(&result); err != nil {
		return nil, fmt.Errorf("reading %s response: %w", cmdType, err)
	}
	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf("%s error %s: %s", cmdType, result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("%s failed", cmdType)
	}

	if len(result.Result) == 0 || result.Result[0] != '{' {
		return nil, nil
	}
	var area Area
	if err := json.Unmarshal(result.Result, &area); err != nil {
		return nil, fmt.Errorf("decoding area response: %w", err)
	}
	return &area, nil
}

type EntityRegistryEntry struct {
	EntityID     string  `json:"entity_id"`
	DeviceID     *string `json:"device_id"`
	Name         *string `json:"name"`
	OriginalName *string `json:"original_name"`
	Platform     string  `json:"platform"`
	AreaID       *string `json:"area_id"`
	DisabledBy   *string `json:"disabled_by"`
	Icon         *string `json:"icon"`
}

func (c *Client) GetEntityRegistry(ctx context.Context) ([]EntityRegistryEntry, error) {
	_, entries, err := c.getRegistries(ctx)
	return entries, err
}

func (c *Client) GetEntityRegistryEntry(ctx context.Context, entityID string) (*EntityRegistryEntry, error) {
	entries, err := c.GetEntityRegistry(ctx)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.EntityID == entityID {
			return &e, nil
		}
	}
	return nil, nil
}

type EntityUpdate struct {
	EntityID   string  `json:"entity_id"`
	Name       *string `json:"name"`
	Icon       *string `json:"icon"`
	AreaID     *string `json:"area_id"`
	DisabledBy *string `json:"disabled_by"`
}

func (c *Client) UpdateEntity(ctx context.Context, update EntityUpdate) (*EntityRegistryEntry, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payload := map[string]any{
		"id":          1,
		"type":        "config/entity_registry/update",
		"entity_id":   update.EntityID,
		"name":        update.Name,
		"icon":        update.Icon,
		"area_id":     update.AreaID,
		"disabled_by": update.DisabledBy,
	}

	if err := conn.WriteJSON(payload); err != nil {
		return nil, fmt.Errorf("sending entity_registry/update: %w", err)
	}

	var result wsResult
	if err := conn.ReadJSON(&result); err != nil {
		return nil, fmt.Errorf("reading entity update response: %w", err)
	}
	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf("entity update error %s: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("entity update failed")
	}

	// HA wraps the entity entry in an "entity_entry" key for this command
	var wrapper struct {
		EntityEntry EntityRegistryEntry `json:"entity_entry"`
	}
	if err := json.Unmarshal(result.Result, &wrapper); err != nil {
		return nil, fmt.Errorf("decoding entity update response: %w", err)
	}
	return &wrapper.EntityEntry, nil
}

func (c *Client) GetDevices(ctx context.Context) ([]Device, error) {
	devices, _, err := c.getRegistries(ctx)
	return devices, err
}

func (c *Client) GetDevice(ctx context.Context, id string) (*Device, error) {
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, nil
}

type DeviceUpdate struct {
	DeviceID    string  `json:"device_id"`
	NameByUser  *string `json:"name_by_user"`
	AreaID      *string `json:"area_id"`
	DisabledBy  *string `json:"disabled_by"`
}

func (c *Client) UpdateDevice(ctx context.Context, update DeviceUpdate) (*Device, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payload := map[string]any{
		"id":          1,
		"type":        "config/device_registry/update",
		"device_id":   update.DeviceID,
		"name_by_user": update.NameByUser,
		"area_id":     update.AreaID,
		"disabled_by": update.DisabledBy,
	}

	if err := conn.WriteJSON(payload); err != nil {
		return nil, fmt.Errorf("sending device_registry/update: %w", err)
	}

	var result wsResult
	if err := conn.ReadJSON(&result); err != nil {
		return nil, fmt.Errorf("reading update response: %w", err)
	}
	if !result.Success {
		if result.Error != nil {
			return nil, fmt.Errorf("device update error %s: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("device update failed")
	}

	var device Device
	if err := json.Unmarshal(result.Result, &device); err != nil {
		return nil, fmt.Errorf("decoding updated device: %w", err)
	}
	return &device, nil
}

func (c *Client) GetDevicesAndEntities(ctx context.Context) ([]Device, []EntityRegistryEntry, error) {
	return c.getRegistries(ctx)
}

func (c *Client) getRegistries(ctx context.Context) ([]Device, []EntityRegistryEntry, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	// Send both commands before reading to pipeline the requests.
	if err := conn.WriteJSON(map[string]any{"id": 1, "type": "config/device_registry/list"}); err != nil {
		return nil, nil, fmt.Errorf("sending device_registry/list: %w", err)
	}
	if err := conn.WriteJSON(map[string]any{"id": 2, "type": "config/entity_registry/list"}); err != nil {
		return nil, nil, fmt.Errorf("sending entity_registry/list: %w", err)
	}

	results := make(map[int]json.RawMessage)
	for i := 0; i < 2; i++ {
		var r wsResult
		if err := conn.ReadJSON(&r); err != nil {
			return nil, nil, fmt.Errorf("reading registry response: %w", err)
		}
		if !r.Success {
			if r.Error != nil {
				return nil, nil, fmt.Errorf("registry request %d error %s: %s", r.ID, r.Error.Code, r.Error.Message)
			}
			return nil, nil, fmt.Errorf("registry request %d failed", r.ID)
		}
		results[r.ID] = r.Result
	}

	var devices []Device
	if err := json.Unmarshal(results[1], &devices); err != nil {
		return nil, nil, fmt.Errorf("decoding devices: %w", err)
	}

	var entities []EntityRegistryEntry
	if err := json.Unmarshal(results[2], &entities); err != nil {
		return nil, nil, fmt.Errorf("decoding entity registry: %w", err)
	}

	return devices, entities, nil
}
