package ffmpeg

import (
	"fmt"
	//"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/stts-se/transtool/protocol"
)

// Chunk2File extracts time chunks from an audio file, creating a subset of files containing "phrases" from the file.
// For initialization, use NewChunk2File().
type Chunk2File struct {
}

// NewChunk2File creates a new Chunk2File after first checking that the ffmpeg command exists
func NewChunk2File() (Chunk2File, error) {
	if err := ffmpegEnabled(); err != nil {
		return Chunk2File{}, err
	}
	return Chunk2File{}, nil
}

// ProcessChunk extracts the specified chunk from the audioFile into the outFile
func (ch Chunk2File) ProcessChunk(audioFile string, chunk protocol.Chunk, outFile, encoding string) error {
	if chunk.Start > chunk.End {
		return fmt.Errorf("cannot process input chunk with negative duration: %v-%v", chunk.Start, chunk.End)
	}
	if chunk.Start == chunk.End && chunk.Start > 0 {
		return fmt.Errorf("cannot process input chunk with zero duration: %v-%v", chunk.Start, chunk.End)
	}

	startFloat := float64(chunk.Start) / 1000.0
	endFloat := float64(chunk.End) / 1000.0
	duration := endFloat - startFloat
	//ffmpeg -y -ss 0 -t 30 -i <in> <out>
	args := []string{"-y", "-ss", fmt.Sprintf("%v", startFloat), "-t", fmt.Sprintf("%v", duration), "-i", audioFile}
	if encoding != "" {
		args = append(args, "-f")
		args = append(args, encoding)
	}
	args = append(args, outFile)
	cmd := exec.Command(FfmpegCmd, args...)
	//log.Printf("chunk2file cmd: %v", cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command %s failed : %#v", cmd, err)
	}
	return nil
}

// Process extracts the specified chunks from the audioFile into folder outDir
func (ch Chunk2File) Process(audioFile string, chunks []protocol.Chunk, outDirName, encoding string) ([]string, error) {
	res := []string{}

	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return res, fmt.Errorf("no such file: %s", audioFile)
	}

	baseDir, fileName := filepath.Split(audioFile)
	ext := filepath.Ext(fileName)
	if encoding != "" {
		ext = encoding
	}

	baseName := strings.TrimSuffix(fileName, ext)
	outDir := filepath.Join(baseDir, outDirName)
	os.MkdirAll(outDir, os.ModePerm)

	for i, chunk := range chunks {
		id := fmt.Sprintf("%04d", i+1)
		outName := fmt.Sprintf("%s_chunk%s%s", baseName, id, ext)
		outFile := filepath.Join(outDir, outName)
		err := ch.ProcessChunk(audioFile, chunk, outFile, encoding)
		if err != nil {
			return res, fmt.Errorf("ProcessChunk failed : %v", err)
		}
		res = append(res, outFile)
	}
	return res, nil
}
