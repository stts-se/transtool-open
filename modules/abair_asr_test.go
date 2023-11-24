package modules

import (
	//"log"
	//"os"
	"path"
	//"path/filepath"
	//"strings"
	"testing"

	"github.com/stts-se/transtool-open/protocol"
)


func init() {
}

func TestAbairASR_UnchunkedWav(t *testing.T) {

	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}
	abairasr, err := NewAbairASR()
	if err != nil {
		t.Errorf("got error from NewAbairASR: %v", err)
		return
	}
	fName := path.Join("test_data", "irish_test1.wav")
	got0, err := abairasr.Process(config, fName, protocol.Chunk{})
	if err != nil {
		t.Errorf("got error from AbairASR.Process: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "baile átha cliath", Chunk: protocol.Chunk{Start: 0, End: 0}},
		}

	got := got0.Chunks
	if len(got) != len(exp) {
		t.Errorf("expected %#v, got %#v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0.Text != exp0.Text {
			t.Errorf("expected text %v, got %v", exp0.Text, got0.Text)
		}

	}
}



func TestAbairASR_ChunkWav(t *testing.T) {
	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}

	abairasr, err := NewAbairASR()
	if err != nil {
		t.Errorf("got error from NewAbairASR: %v", err)
		return
	}
	fName := path.Join("test_data", "irish_test1.wav")
	got0, err := abairasr.Process(config, fName, protocol.Chunk{Start: 0, End: 500})
	if err != nil {
		t.Errorf("got error from AbairASR.Process: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "baile", Chunk: protocol.Chunk{Start: 0, End: 0}},
	}

	got := got0.Chunks
	if len(got) != len(exp) {
		t.Errorf("expected %#v, got %#v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}

