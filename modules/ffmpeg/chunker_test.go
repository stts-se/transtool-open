package ffmpeg

import (
	"path"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

func abs(diff int64) int64 {
	if diff > 0 {
		return diff
	}
	return -diff
}

// ffmpeg works differently on different machines, so we won't necessary get the exact same figures
const testDiffMargin = int64(50)

func approxEqual(t1, t2 protocol.Chunk) bool {
	if (abs(t1.Start - t2.Start)) > testDiffMargin {
		return false
	}
	if (abs(t1.End - t2.End)) > testDiffMargin {
		return false
	}
	return true
}

func TestChunkerMP3(t *testing.T) {
	chunker, err := NewDefaultChunker()
	if err != nil {
		t.Errorf("got error from NewFfmpegChunker: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	got, err := chunker.ProcessFile(fName)
	if err != nil {
		t.Errorf("got error from FfmpegChunker.Process: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 0, End: 1894},
		{Start: 2441, End: 4210},
		{Start: 4957, End: 8310},
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v +-%d, got %v", exp, testDiffMargin, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if !approxEqual(got0, exp0) {
			t.Errorf("expected %v +-%d, got %v", exp0, testDiffMargin, got0)
		}

	}
}

func TestInnerChunkerMP3(t *testing.T) {
	chunker, err := NewChunker(1000, 100)
	if err != nil {
		t.Errorf("got error from NewFfmpegChunker: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	got, err := chunker.ProcessChunk(fName, protocol.Chunk{Start: 2441, End: 8310})
	if err != nil {
		t.Errorf("got error from FfmpegChunker.Process: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 2565, End: 4038},
		{Start: 5099, End: 8310},
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v +-%d, got %v", exp, testDiffMargin, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if !approxEqual(got0, exp0) {
			t.Errorf("expected %v +-%d, got %v", exp0, testDiffMargin, got0)
		}

	}
}

func TestChunkerWav(t *testing.T) {
	chunker, err := NewDefaultChunker()
	if err != nil {
		t.Errorf("got error from NewFfmpegChunker: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	got, err := chunker.ProcessFile(fName)
	if err != nil {
		t.Errorf("got error from FfmpegChunker.Process: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 0, End: 1894},
		{Start: 2441, End: 4210},
		{Start: 4957, End: 8270},
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v +-%d, got %v", exp, testDiffMargin, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if !approxEqual(got0, exp0) {
			t.Errorf("expected %v +-%d, got %v", exp0, testDiffMargin, got0)
		}

	}
}

func TestChunkerWavTruncEnd(t *testing.T) {
	chunker, err := NewDefaultChunker()
	if err != nil {
		t.Errorf("got error from NewFfmpegChunker: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences_trunc_end.wav")
	got, err := chunker.ProcessFile(fName)
	if err != nil {
		t.Errorf("got error from FfmpegChunker.Process: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 0, End: 1459},
		{Start: 2005, End: 3774},
		{Start: 4522, End: 6740},
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v +-%d, got %v", exp, testDiffMargin, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if !approxEqual(got0, exp0) {
			t.Errorf("expected %v +-%d, got %v", exp0, testDiffMargin, got0)
		}

	}
}
