package ffprobe

import (
	"fmt"
	"os/exec"
)

var FFProbeCmd = "ffprobe"

func ffprobeEnabled() error {
	_, pErr := exec.LookPath(FFProbeCmd)
	if pErr != nil {
		return fmt.Errorf("external command does not exist: %s", FFProbeCmd)
	}
	return nil
}
