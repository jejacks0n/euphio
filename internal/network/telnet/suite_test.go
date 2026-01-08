package telnet_test

import (
	"log/slog"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"euphio/internal/app"
)

func TestTelnet(t *testing.T) {
	RegisterFailHandler(Fail)

	// Initialize logger for tests
	// We can use a text handler writing to stdout, or io.Discard if we want silence
	app.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	RunSpecs(t, "Telnet Suite")
}
