package client_test

import (
	"context"
	"testing"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/client"
	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/testutil"
)

const testToken = "test-token-abc123"

func TestVerify_success(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: testToken})
	c := client.New(srv.URL, testToken)

	if err := c.Verify(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestVerify_invalidToken(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: testToken})
	c := client.New(srv.URL, "wrong-token")

	err := c.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestGetDevices_empty(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: testToken})
	c := client.New(srv.URL, testToken)

	devices, err := c.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
}

func TestGetDevices_returnsDevices(t *testing.T) {
	mac := "aa:bb:cc:dd:ee:ff"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: testToken,
		Devices: []testutil.MockDevice{
			{
				ID:           "device-1",
				Name:         "Living Room Light",
				Manufacturer: testutil.Ptr("Philips"),
				Model:        testutil.Ptr("Hue Bulb"),
				Connections:  [][]string{{"mac", mac}},
				Identifiers:  [][]string{{"hue", "hue-light-1"}},
			},
			{
				ID:   "device-2",
				Name: "Temperature Sensor",
				Identifiers: [][]string{{"zha", "00:12:4b:00:1a:2b:3c:4d"}},
			},
		},
	})
	c := client.New(srv.URL, testToken)

	devices, err := c.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	d := devices[0]
	if d.ID != "device-1" {
		t.Errorf("expected id device-1, got %s", d.ID)
	}
	if d.Name != "Living Room Light" {
		t.Errorf("expected name Living Room Light, got %s", d.Name)
	}
	if len(d.Connections) != 1 || d.Connections[0][1] != mac {
		t.Errorf("expected connection mac %s, got %v", mac, d.Connections)
	}
	if len(d.Identifiers) != 1 || d.Identifiers[0][0] != "hue" {
		t.Errorf("expected hue identifier, got %v", d.Identifiers)
	}
}

func TestGetStates_returnsEntities(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: testToken,
		EntityStates: []testutil.MockEntityState{
			{EntityID: "light.living_room", State: "on", Attributes: map[string]any{"brightness": 255}},
			{EntityID: "sensor.temperature", State: "21.5"},
		},
	})
	c := client.New(srv.URL, testToken)

	states, err := c.GetStates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[0].EntityID != "light.living_room" || states[0].State != "on" {
		t.Errorf("unexpected first state: %+v", states[0])
	}
}

func TestGetDevices_withEntityRegistry(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: testToken,
		Devices: []testutil.MockDevice{
			{ID: deviceID, Name: "ESP32 Board"},
		},
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.living_room_temperature", DeviceID: &deviceID, OriginalName: testutil.Ptr("Living Room Temperature"), Platform: "esphome"},
			{EntityID: "sensor.living_room_humidity", DeviceID: &deviceID, OriginalName: testutil.Ptr("Living Room Humidity"), Platform: "esphome"},
		},
	})
	c := client.New(srv.URL, testToken)

	devices, entities, err := c.GetDevicesAndEntities(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if len(entities) != 2 {
		t.Fatalf("expected 2 entity registry entries, got %d", len(entities))
	}
	if entities[0].Platform != "esphome" {
		t.Errorf("expected platform esphome, got %s", entities[0].Platform)
	}
}

func TestGetStates_invalidToken(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: testToken})
	c := client.New(srv.URL, "wrong-token")

	// Verify will fail first during provider Configure, but GetStates uses REST
	// so test it directly here
	_, err := c.GetStates(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}
