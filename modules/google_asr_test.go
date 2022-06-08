package modules

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

var googleCredentials = path.Join("..", "credentials.json")

func init() {
	if _, err := os.Stat(googleCredentials); os.IsNotExist(err) {
		path, err := filepath.Abs(googleCredentials)
		if err != nil {
			path = googleCredentials
		}
		log.Fatalf("No credentials file: %s", path)
	}
}

func TestGoogleASR_UnchunkedWav(t *testing.T) {

	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}
	googler, err := NewGoogleASR(googleCredentials)
	if err != nil {
		t.Errorf("got error from NewGoogleASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.wav")
	got0, err := googler.Process(config, fName, protocol.Chunk{})
	if err != nil {
		t.Errorf("got error from GoogleASR.ProcessFile: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en", Chunk: protocol.Chunk{Start: 400, End: 1200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 1200, End: 1700}},
		{Text: "en", Chunk: protocol.Chunk{Start: 1700, End: 3100}},
		{Text: "annan", Chunk: protocol.Chunk{Start: 3100, End: 3600}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 3600, End: 4000}},
		{Text: "och", Chunk: protocol.Chunk{Start: 4000, End: 5600}},
		{Text: "en", Chunk: protocol.Chunk{Start: 5600, End: 6000}},
		{Text: "tredje", Chunk: protocol.Chunk{Start: 6000, End: 6800}},
		{Text: "sista", Chunk: protocol.Chunk{Start: 6800, End: 7200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 7200, End: 7900}},
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

func TestGoogleASR_UnchunkedOpus(t *testing.T) {

	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "opus",
		SampleRate:   48000,
		ChannelCount: 2,
	}
	googler, err := NewGoogleASR(googleCredentials)
	if err != nil {
		t.Errorf("got error from NewGoogleASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.opus")
	got0, err := googler.Process(config, fName, protocol.Chunk{})
	if err != nil {
		t.Errorf("got error from GoogleASR.ProcessFile: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en", Chunk: protocol.Chunk{Start: 400, End: 1200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 1200, End: 1700}},
		{Text: "en", Chunk: protocol.Chunk{Start: 1700, End: 3100}},
		{Text: "annan", Chunk: protocol.Chunk{Start: 3100, End: 3600}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 3600, End: 4000}},
		{Text: "och", Chunk: protocol.Chunk{Start: 4000, End: 5600}},
		{Text: "en", Chunk: protocol.Chunk{Start: 5600, End: 6000}},
		{Text: "tredje", Chunk: protocol.Chunk{Start: 6000, End: 6800}},
		{Text: "sista", Chunk: protocol.Chunk{Start: 6800, End: 7200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 7200, End: 7900}},
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

func TestGoogleASR_ChunkWav(t *testing.T) {
	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "wav",
		SampleRate:   44100,
		ChannelCount: 1,
	}
	googler, err := NewGoogleASR(googleCredentials)
	if err != nil {
		t.Errorf("got error from NewGoogleASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.wav")
	got0, err := googler.Process(config, fName, protocol.Chunk{Start: 2397, End: 4172})
	if err != nil {
		t.Errorf("got error from GoogleASR.ProcessFile: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en", Chunk: protocol.Chunk{Start: 0, End: 700}},
		{Text: "annan", Chunk: protocol.Chunk{Start: 700, End: 1200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 1200, End: 1600}},
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

func TestGoogleASR_ChunkOpus(t *testing.T) {
	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "opus",
		SampleRate:   48000,
		ChannelCount: 2,
	}
	googler, err := NewGoogleASR(googleCredentials)
	if err != nil {
		t.Errorf("got error from NewGoogleASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.opus")
	got0, err := googler.Process(config, fName, protocol.Chunk{Start: 2397, End: 4172})
	if err != nil {
		t.Errorf("got error from GoogleASR.ProcessFile: %v", err)
		return
	}
	exp := []protocol.ASROutputChunk{
		{Text: "en", Chunk: protocol.Chunk{Start: 0, End: 700}},
		{Text: "annan", Chunk: protocol.Chunk{Start: 700, End: 1200}},
		{Text: "mening", Chunk: protocol.Chunk{Start: 1200, End: 1600}},
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

func TestGoogleASR_NegativeChunk(t *testing.T) {
	var err error
	var expErr string
	config := protocol.ASRConfig{
		Lang:         "sv-SE",
		Encoding:     "opus",
		SampleRate:   48000,
		ChannelCount: 2,
	}
	googler, err := NewGoogleASR(googleCredentials)
	if err != nil {
		t.Errorf("got error from NewGoogleASR: %v", err)
		return
	}
	fName := path.Join("test_data", "three_sentences.opus")

	// negative duration
	_, err = googler.Process(config, fName, protocol.Chunk{Start: 2397, End: 2396})
	if err == nil {
		t.Errorf("expected error from GoogleASR.ProcessFile")
		return
	}
	expErr = "cannot process input chunk with negative duration"
	if !strings.Contains(err.Error(), expErr) {
		t.Errorf("expected error from GoogleASR.ProcessFile to match %s, found %v", expErr, err)
		return
	}

	// zero duration
	_, err = googler.Process(config, fName, protocol.Chunk{Start: 2397, End: 2397})
	if err == nil {
		t.Errorf("expected error from GoogleASR.ProcessFile")
		return
	}
	expErr = "cannot process input chunk with zero duration"
	if !strings.Contains(err.Error(), expErr) {
		t.Errorf("expected error from GoogleASR.ProcessFile to match %s, found %v", expErr, err)
		return
	}
}
