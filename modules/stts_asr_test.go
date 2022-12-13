package modules

import (
	//"log"
	//"os"
	"path"
	//"path/filepath"
	//"strings"
	"testing"

	"github.com/stts-se/transtool/protocol"
)


func init() {
}

func TestSttsASR_UnchunkedWav(t *testing.T) {

	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}
	sttsasr, err := NewSttsASR()
	if err != nil {
		t.Errorf("got error from NewSttsASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.wav")
	got0, err := sttsasr.Process(config, fName, protocol.Chunk{})
	if err != nil {
		t.Errorf("got error from SttsASR.Process: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en mening en annan mening och en tredje sista mening", Chunk: protocol.Chunk{Start: 0, End: 0}},
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



func TestSttsASR_ChunkWav(t *testing.T) {
	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}

	sttsasr, err := NewSttsASR()
	if err != nil {
		t.Errorf("got error from NewSttsASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.wav")
	got0, err := sttsasr.Process(config, fName, protocol.Chunk{Start: 600, End: 2000})
	if err != nil {
		t.Errorf("got error from SttsASR.Process: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en mening", Chunk: protocol.Chunk{Start: 0, End: 0}},
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

