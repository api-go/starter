package starter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/api-go/file"
	"github.com/api-go/starter"
)

func TestBuiltinLogCommandSubprocess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.log")
	if err := file.Write(path, "first\nsecond\nthird\n"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBuiltinLogHelper$")
	cmd.Env = append(os.Environ(), "APIGO_LOG_COMMAND_FIXTURE="+path)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, data)
	}
	if !strings.Contains(string(data), "second\nthird\n") || strings.Contains(string(data), "first") {
		t.Fatalf("bad output: %s", data)
	}
}

func TestBuiltinLogHelper(t *testing.T) {
	path := os.Getenv("APIGO_LOG_COMMAND_FIXTURE")
	if path == "" {
		return
	}
	os.Args = []string{os.Args[0], "log", "-tail", "2", path}
	if err := starter.Start(); err != nil {
		t.Fatal(err)
	}
}
