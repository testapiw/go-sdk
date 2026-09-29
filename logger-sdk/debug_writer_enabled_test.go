//go:build debug

package logger

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestDebugSendsStructuredUDPDatagram(t *testing.T) {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOG_SERVICE", "logger-test")

	instance, err := New(Config{
		Level: LevelDebug, LogDir: t.TempDir(), FilePrefix: "test",
		DebugUDPAddr: listener.LocalAddr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	instance.Debug("tick.started", map[string]any{"tick_id": 123, "coins": 5})

	buffer := make([]byte, 2048)
	n, _, err := listener.ReadFromUDP(buffer)
	if err != nil {
		t.Fatal(err)
	}
	var packet debugPacket
	if err := json.Unmarshal(buffer[:n], &packet); err != nil {
		t.Fatal(err)
	}
	if packet.Level != "DEBUG" || packet.Service != "logger-test" || packet.Message != "tick.started" {
		t.Fatalf("unexpected packet metadata: %#v", packet)
	}
	var data []map[string]any
	if err := json.Unmarshal(packet.Data, &data); err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 || data[0]["tick_id"] != float64(123) || data[0]["coins"] != float64(5) {
		t.Fatalf("unexpected packet data: %#v", data)
	}
}
