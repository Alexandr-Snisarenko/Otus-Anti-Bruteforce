package version

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	Release   = "UNKNOWN"
	BuildDate = "UNKNOWN"
	GitHash   = "UNKNOWN"
)

func PrintVersion() {
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Release   string
		BuildDate string
		GitHash   string
	}{
		Release:   Release,
		BuildDate: BuildDate,
		GitHash:   GitHash,
	}); err != nil {
		fmt.Printf("error while decode version info: %v\n", err)
	}
}
