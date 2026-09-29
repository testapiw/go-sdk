//go:build !debug

package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDebugIsCompiledOut(t *testing.T) {
	dir := t.TempDir()
	instance, err := New(Config{Level: LevelDebug, LogDir: dir, FilePrefix: "test"})
	if err != nil {
		t.Fatal(err)
	}
	instance.Debug("should.not.appear", map[string]any{"value": 1})
	if err := instance.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "test-"+time.Now().Format("2006-01-02")+".log")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 0 {
		t.Fatalf("debug log was not compiled out: %q", content)
	}
}
