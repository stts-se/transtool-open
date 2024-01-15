package modules

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

const testDiffMargin = int64(50)

//var wikispeechAnnotatorTestLocation string

func abs(diff int64) int64 {
	if diff > 0 {
		return diff
	}
	return -diff
}

func approxEqual(t1, t2 protocol.Chunk) bool {
	if (abs(t1.Start - t2.Start)) > testDiffMargin {
		return false
	}
	if (abs(t1.End - t2.End)) > testDiffMargin {
		return false
	}
	return true
}

func init() {
	envName := "WIKISPEECH_ANNOTATOR"
	val := os.Getenv(envName)
	if val == "" {
		log.Fatalf("[wikispeech_annotator] Environment variable %s must be set (location for local copy of https://github.com/stts-se/wikispeech-annotator)", envName)
	}
	WikispeechAnnotatorDir = val
}

func TestWikispeechAnnotatorMP3(t *testing.T) {

	anno, err := NewWikispeechAnnotator()
	if err != nil {
		t.Errorf("got error from NewWikispeechAnnotator: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences_48k.mp3")
	got, err := anno.VAD(fName)
	if err != nil {
		t.Errorf("got error from annotator.VAD: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 790, End: 1840},
		{Start: 2720, End: 4200},
		{Start: 5240, End: 7970},
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

func TestWikispeechAnnotatorWAV(t *testing.T) {

	anno, err := NewWikispeechAnnotator()
	if err != nil {
		t.Errorf("got error from NewWikispeechAnnotator: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences_48k.wav")
	got, err := anno.VAD(fName)
	if err != nil {
		t.Errorf("got error from annotator.VAD: %v", err)
		return
	}
	exp := []protocol.Chunk{
		{Start: 790, End: 1840},
		{Start: 2720, End: 4200},
		{Start: 5240, End: 7970},
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
