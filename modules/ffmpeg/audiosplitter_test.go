package ffmpeg

import (
	"os"
	"path"
	"testing"
)

func TestAudioSplitMP3(t *testing.T) {
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunker, err := NewDefaultChunker()
	if err != nil {
		t.Errorf("got error from NewChunker: %v", err)
		return
	}
	c2f, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}

	chunks, err := chunker.ProcessFile(fName)
	if err != nil {
		t.Errorf("got error from Chunker.Process: %v", err)
		return
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-audiosplit-test-1-mp3"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := c2f.Process(fName, chunks, tmpBase, "")
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

func TestAudioSplitWav(t *testing.T) {
	chunker, err := NewDefaultChunker()
	if err != nil {
		t.Errorf("got error from Chunker: %v", err)
		return
	}
	c2f, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")

	chunks, err := chunker.ProcessFile(fName)
	if err != nil {
		t.Errorf("got error from Chunker.Process: %v", err)
		return
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-audiosplit-test-1-wav"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := c2f.Process(fName, chunks, tmpBase, "")
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
