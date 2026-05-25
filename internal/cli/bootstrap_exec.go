package cli

import (
	"fmt"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func executeBootstrap(targetPath string, withWarnPolicy bool) error {
	err := sensors.BootstrapRepoWithPolicy(targetPath, withWarnPolicy)
	if err != nil {
		logf(LogLevelError, "[ERROR] Bootstrap failed: %v\n", err)
		return fmt.Errorf("bootstrap failed: %v", err)
	}
	return nil
}
