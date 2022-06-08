package ffmpeg

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/stts-se/transtool/protocol"
)

// ChunkExtractor extracts time chunks from an audio file, creating a subset of "phrases" from the file.
// For initialization, use NewChunkExtractor().
type ChunkExtractor struct {
	chunk2file Chunk2File
}

// NewChunkExtractor creates a new ChunkExtractor after first checking that the ffmpeg command exists
func NewChunkExtractor() (ChunkExtractor, error) {
	c2f, err := NewChunk2File()
	if err != nil {
		return ChunkExtractor{}, err
	}
	return ChunkExtractor{chunk2file: c2f}, nil
}

// ProcessFileWithContext an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFileWithContext(payload protocol.SplitRequestPayload, encoding string) (protocol.AnnotationWithAudioData, error) {
	if _, err := os.Stat(payload.Audio); os.IsNotExist(err) {
		return protocol.AnnotationWithAudioData{}, fmt.Errorf("no such file: %s", payload.Audio)
	}

	offset := payload.Chunk.Start - payload.LeftContext
	if offset < 0 {
		offset = 0
	}
	processChunk := protocol.Chunk{
		Start: offset,
		End:   payload.Chunk.End + payload.RightContext,
	}

	ext := filepath.Ext(payload.Audio)
	ext = strings.TrimPrefix(ext, ".")
	ext = trimURLParamsRE.ReplaceAllString(ext, "")
	if encoding == "" {
		encoding = ext
	}
	btss, err := ch.ProcessFile(payload.Audio, []protocol.Chunk{processChunk}, encoding)
	if err != nil {
		return protocol.AnnotationWithAudioData{}, err
	}

	if len(btss) != 1 {
		return protocol.AnnotationWithAudioData{}, fmt.Errorf("expected one byte array, found %d", len(btss))
	}

	bts := btss[0]

	//os.WriteFile("chunk_extractor_debug.wav", bts, 0644)

	res := protocol.AnnotationWithAudioData{
		Base64Audio: base64.StdEncoding.EncodeToString(bts),
		FileType:    encoding,
		Offset:      offset,
	}
	res.Start = payload.Chunk.Start - offset
	res.End = payload.Chunk.End - offset
	return res, nil
}

var trimURLParamsRE = regexp.MustCompile(`\?[^?.]*$`)

// ProcessFile an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFile(audioFile string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
	res := [][]byte{}
	// if _, err := os.Stat(audioFile); os.IsNotExist(err) {
	// 	return res, fmt.Errorf("No such file: %s", audioFile)
	// }
	for _, chunk := range chunks {

		if chunk.Start > chunk.End {
			return res, fmt.Errorf("cannot process input chunk with negative duration: %v-%v", chunk.Start, chunk.End)
		}
		if chunk.Start == chunk.End && chunk.Start > 0 {
			return res, fmt.Errorf("cannot process input chunk with zero duration: %v-%v", chunk.Start, chunk.End)
		}

		ext := filepath.Ext(audioFile)
		ext = strings.TrimPrefix(ext, ".")
		ext = trimURLParamsRE.ReplaceAllString(ext, "")
		if encoding != "" {
			ext = encoding
		} else {
			encoding = ext
		}
		id, err := uuid.NewUUID()
		if err != nil {
			return res, fmt.Errorf("couldn't create uuid : %v", err)
		}
		tmpFile := path.Join(os.TempDir(), fmt.Sprintf("chunk-extractor-%s.%s", id, ext))
		//log.Info("chunk_extractor tmpFile", tmpFile)
		defer os.Remove(tmpFile)
		//c2fStart := time.Now()
		err = ch.chunk2file.ProcessChunk(audioFile, chunk, tmpFile, encoding)
		if err != nil {
			return res, fmt.Errorf("chunk2file.ProcessChunk failed : %v", err)
		}
		//c2fDur := time.Since(c2fStart)
		//log.Info("chunk2file dur %v", c2fDur)
		bytes, err := os.ReadFile(tmpFile)
		if err != nil {
			return res, fmt.Errorf("failed to read file : %v", err)
		}
		res = append(res, bytes)
	}
	return res, nil
}
