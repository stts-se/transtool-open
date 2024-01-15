package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/stts-se/transtool/protocol"
)

var WikispeechAnnotatorDir string

// WikispeechAnnotator is used to split audio files on silence, creating a subset of "phrases" from the file.
type WikispeechAnnotator struct {
}

type VADChunk struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  int     `json:"text"`
}

type VADJSON struct {
	Chunks []VADChunk `json:"vad"`
}

func wsAnnotatorEnabled() error {
	if WikispeechAnnotatorDir == "" {
		return fmt.Errorf("annotator folder is not set")
	}
	_, err := os.Stat(WikispeechAnnotatorDir)
	if err != nil {
		return fmt.Errorf("annotator folder does not exist: %s", WikispeechAnnotatorDir)
	}
	return nil
}

// NewWikispeechAnnotator creates a new WikispeechAnnotator after first checking that the application folder exists
func NewWikispeechAnnotator() (WikispeechAnnotator, error) {
	if err := wsAnnotatorEnabled(); err != nil {
		return WikispeechAnnotator{}, err
	}
	cmd := exec.Command("python3", "annotator.py", "--help")
	cmd.Dir = WikispeechAnnotatorDir
	_, err := cmd.CombinedOutput()
	if err != nil {
		return WikispeechAnnotator{}, fmt.Errorf("annotator.py command cannot be run : %v", err)
	}
	return WikispeechAnnotator{}, nil
}

// VAD splits the audioFile into voiced chunks
func (wa WikispeechAnnotator) VAD(audioFile string) ([]protocol.Chunk, error) {
	res := []protocol.Chunk{}

	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return res, fmt.Errorf("no such file: %s", audioFile)
	}
	format := "PCM"
	if strings.HasSuffix(strings.ToLower(audioFile), "mp3") {
		format = "MP3"
	} else if strings.HasSuffix(strings.ToLower(audioFile), "ogg") {
		format = "OGG"
	} else if strings.HasSuffix(strings.ToLower(audioFile), "opus") {
		format = "OPUS"
	} else if strings.HasSuffix(strings.ToLower(audioFile), "wav") {
		format = "PCM"
	} else {
		return res, fmt.Errorf("failed to retrieve file typ for input file %s", audioFile)
	}
	audioFileAbs, err := filepath.Abs(audioFile)
	if err != nil {
		return res, fmt.Errorf("failed to find absolute path for input file %s : %v", audioFile, err)
	}

	//HB testing 2201 annotator command can be "vad" or "bk"
	annotatorCommand := "vad"
	//annotatorCommand := "bk"

	
	cmdArgs := []string{"annotator.py", annotatorCommand,
		fmt.Sprintf("--audioinputformat=%s", format),
		"--audioinputtype=FILE",
		"--returntype=JSON",
		audioFileAbs}
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = WikispeechAnnotatorDir
	out, err := cmd.Output()
	if err != nil {
		return res, fmt.Errorf("command python3 %s failed : %v", strings.Join(cmdArgs, " "), err)
	}
	outS := strings.TrimSpace(string(out))
	fmt.Println(outS)

	
	var vadJSON VADJSON
	err = json.Unmarshal([]byte(outS), &vadJSON)
	if err != nil {
		return res, fmt.Errorf("failed to unmarshal vad output : %v", err)
	}

	for _, vc := range vadJSON.Chunks {
		chunk := protocol.Chunk{
			Start: int64(vc.Start * 1000.0),
			End:   int64(vc.End * 1000.0),
		}
		res = append(res, chunk)
	}

	return res, nil
}
