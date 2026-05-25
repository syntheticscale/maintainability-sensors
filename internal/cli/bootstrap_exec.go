package cli

import (
	"fmt"
	"os"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func executeBootstrap(targetPath string, withWarnPolicy bool) {
	err := sensors.BootstrapRepoWithPolicy(targetPath, withWarnPolicy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Bootstrap failed: %v\n", err)
		os.Exit(1)
	}
}
