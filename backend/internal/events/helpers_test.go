package events

import (
	"os"

	"github.com/rs/zerolog"
)

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.Disabled)
}
