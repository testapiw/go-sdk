//go:build debug

package logger

import (
	"encoding/json"
	"net"
	"os"
	"time"
)

type debugWriter interface {
	write(time.Time, string, string)
	close() error
}

type udpDebugWriter struct {
	conn    *net.UDPConn
	service string
}

type debugPacket struct {
	Time    time.Time       `json:"time"`
	Service string          `json:"service"`
	Level   string          `json:"level"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func newDebugUDPWriter(addr string) (debugWriter, error) {
	remote, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return nil, err
	}
	return &udpDebugWriter{conn: conn, service: os.Getenv("LOG_SERVICE")}, nil
}

func (w *udpDebugWriter) write(at time.Time, message, data string) {
	packet, err := json.Marshal(debugPacket{
		Time: at.UTC(), Service: w.service, Level: LevelDebug.String(),
		Message: message, Data: json.RawMessage(data),
	})
	if err != nil {
		return
	}
	_, _ = w.conn.Write(packet)
}

func (w *udpDebugWriter) close() error { return w.conn.Close() }
