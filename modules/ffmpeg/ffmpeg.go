package ffmpeg

import (
	"fmt"
	"os/exec"
)

var FfmpegCmd = "ffmpeg"

func ffmpegEnabled() error {
	_, pErr := exec.LookPath(FfmpegCmd)
	if pErr != nil {
		return fmt.Errorf("external command does not exist: %s", FfmpegCmd)
	}
	return nil
}
