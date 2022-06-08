package ffmpeg

import (
	"os"
	"path"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

func TestChunk2File1MP3(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-1-mp3"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join("../test_data", tmpBase, "three_sentences_chunk0001.mp3"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0002.mp3"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0003.mp3"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}

func TestChunk2File1Wav(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1600},
		{Start: 1600, End: 3922},
		{Start: 3922, End: 7684},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-1-wav"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join("../test_data", tmpBase, "three_sentences_chunk0001.wav"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0002.wav"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0003.wav"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}
