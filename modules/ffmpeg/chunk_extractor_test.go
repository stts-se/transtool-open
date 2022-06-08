package ffmpeg

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

func TestChunkExtractorFileMP3(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessFile(fName, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{13183, 18826, 30738} // approximate byte len
	if len(got) != len(expLens) {
		t.Errorf("expected %v, got %v", expLens, got)
		return
	}
	for i, exp0 := range expLens {
		got0 := got[i]
		max := exp0 + 200
		min := exp0 - 200
		if len(got0) < min || len(got0) > max {
			t.Errorf("expected a value between %v and %v, got %v", min, max, len(got0))
		}

	}
}

func TestChunkExtractorFileWAV(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessFile(fName, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{140052, 202762, 331886} // approximate byte len
	if len(got) != len(expLens) {
		t.Errorf("expected %v, got %v", expLens, got)
		return
	}
	for i, exp0 := range expLens {
		got0 := got[i]
		max := exp0 + 200
		min := exp0 - 200
		if len(got0) < min || len(got0) > max {
			t.Errorf("expected a value between %v and %v, got %v", min, max, len(got0))
		}

	}
}

func TestChunkExtractorURLWAV(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("got error from Getwd: %v", err)
	}
	url := fmt.Sprintf("file://%s/../test_data/three_sentences.wav", wd)
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessFile(url, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{140052, 202762, 331886} // approximate byte len
	if len(got) != len(expLens) {
		t.Errorf("expected %v, got %v", expLens, got)
		return
	}
	for i, exp0 := range expLens {
		got0 := got[i]
		max := exp0 + 200
		min := exp0 - 200
		if len(got0) < min || len(got0) > max {
			t.Errorf("expected a value between %v and %v, got %v", min, max, len(got0))
		}

	}
}

func TestChunkExtractorFileWithContext(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	file := "../test_data/three_sentences.wav"
	leftContext := int64(100)
	rightContext := int64(100)
	var chunk protocol.Chunk
	var got protocol.AnnotationWithAudioData

	//
	chunk = protocol.Chunk{Start: 165, End: 261}

	request := protocol.SplitRequestPayload{Audio: file, Chunk: chunk, LeftContext: leftContext, RightContext: rightContext}
	got, err = chunker.ProcessFileWithContext(request, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Start)
	}
	if got.End != 196 {
		t.Errorf("expected %v, got %v", 196, got.End)
	}

	//
	request.Chunk = protocol.Chunk{Start: 405, End: 514}

	got, err = chunker.ProcessFileWithContext(request, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Start)
	}
	if got.End != 209 {
		t.Errorf("expected %v, got %v", 209, got.End)
	}

	//
	request.Chunk = protocol.Chunk{Start: 405, End: 514}

	got, err = chunker.ProcessFileWithContext(request, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Start)
	}
	if got.End != 209 {
		t.Errorf("expected %v, got %v", 209, got.End)
	}

	//
	request.Chunk = protocol.Chunk{Start: 767, End: 826}

	got, err = chunker.ProcessFileWithContext(request, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Start)
	}
	if got.End != 159 {
		t.Errorf("expected %v, got %v", 159, got.End)
	}
}
