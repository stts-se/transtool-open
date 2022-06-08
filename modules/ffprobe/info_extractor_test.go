package ffprobe

import (
	"encoding/json"
	"path"
	"testing"
)

func TestFFprobeInfo1(t *testing.T) {
	ff := ffprobeInfo{
		Streams: []ffprobeStream{
			{
				CodecName:     "pcm_s16le",
				CodecLongName: "PCM signed 16-bit little-endian",
				//CodecType:     "audio",
				SampleRate:    "44100",
				Channels:      1,
				BitsPerSample: 16,
				Duration:      "8.265420",
				BitRate:       "705600"},
		},
		Format: ffprobeFormat{
			FileName: "../test_data/three_sentences.wav",
			Streams:  1,
			//"nb_programs": 0,
			FormatName:     "wav",
			FormatLongName: "WAV / WAVE (Waveform Audio)",
			Duration:       "8.265420",
			Size:           "729054",
			BitRate:        "705642",
			//"probe_score": 99
		},
	}
	_, err := json.MarshalIndent(ff, " ", " ")
	if err != nil {
		t.Errorf("Got error from json.Marshal: %v", err)
	}
}

func TestFFprobeInfo2(t *testing.T) {
	js := `{
		"streams": [
			{
				"index": 0,
				"codec_name": "pcm_s16le",
				"codec_long_name": "PCM signed 16-bit little-endian",
				"codec_type": "audio",
				"codec_time_base": "1/44100",
				"codec_tag_string": "[1][0][0][0]",
				"codec_tag": "0x0001",
				"sample_fmt": "s16",
				"sample_rate": "44100",
				"channels": 1,
				"bits_per_sample": 16,
				"r_frame_rate": "0/0",
				"avg_frame_rate": "0/0",
				"time_base": "1/44100",
				"duration_ts": 364505,
				"duration": "8.265420",
				"bit_rate": "705600",
				"disposition": {
					"default": 0,
					"dub": 0,
					"original": 0,
					"comment": 0,
					"lyrics": 0,
					"karaoke": 0,
					"forced": 0,
					"hearing_impaired": 0,
					"visual_impaired": 0,
					"clean_effects": 0,
					"attached_pic": 0
				}
			}
		],
		"format": {
			"filename": "../test_data/three_sentences.wav",
			"nb_streams": 1,
			"nb_programs": 0,
			"format_name": "wav",
			"format_long_name": "WAV / WAVE (Waveform Audio)",
			"duration": "8.265420",
			"size": "729054",
			"bit_rate": "705642",
			"probe_score": 99
		}
	}
	`
	var ff ffprobeInfo
	err := json.Unmarshal([]byte(js), &ff)
	if err != nil {
		t.Errorf("json.Unmarshal failed: %v", err)
	}
}

func TestInfoExtractorWav(t *testing.T) {
	ie, err := NewInfoExtractor()
	if err != nil {
		t.Errorf("got error from NewInfoExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	got, err := ie.Process(fName)
	if err != nil {
		t.Errorf("got error from InfoExtractor.Process: %v", err)
		return
	}

	exp := AudioInfo{
		FileName:      "../test_data/three_sentences.wav",
		ChannelCount:  1,
		StreamCount:   1,
		SampleRate:    44100,
		Duration:      8265,
		SampleCount:   364505,
		FileSize:      729054,
		BitRate:       705642,
		BitsPerSample: 16,
		Codec:         "pcm_s16le",
		CodecLong:     "PCM signed 16-bit little-endian",
		Format:        "wav",
		FormatLong:    "WAV / WAVE (Waveform Audio)",
	}

	if exp != got {
		t.Errorf("Expected\n%#v\n got\n%#v", exp, got)
	}
}

func TestInfoExtractorMP3(t *testing.T) {
	ie, err := NewInfoExtractor()
	if err != nil {
		t.Errorf("got error from NewInfoExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	got, err := ie.Process(fName)
	if err != nil {
		t.Errorf("got error from InfoExtractor.Process: %v", err)
		return
	}

	exp := AudioInfo{
		FileName:      "../test_data/three_sentences.mp3",
		ChannelCount:  1,
		StreamCount:   1,
		SampleRate:    44100,
		Duration:      8307,
		SampleCount:   117227520,
		FileSize:      82781,
		BitRate:       79722,
		BitsPerSample: 0,
		Codec:         "mp3",
		CodecLong:     "MP3 (MPEG audio layer 3)",
		Format:        "mp3",
		FormatLong:    "MP2/3 (MPEG audio layer 2/3)",
	}

	if exp != got {
		t.Errorf("Expected\n%#v\n got\n%#v", exp, got)
	}
}

func TestInfoExtractorOpus(t *testing.T) {
	ie, err := NewInfoExtractor()
	if err != nil {
		t.Errorf("got error from NewInfoExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.opus")
	got, err := ie.Process(fName)
	if err != nil {
		t.Errorf("got error from InfoExtractor.Process: %v", err)
		return
	}

	exp := AudioInfo{
		FileName:      "../test_data/three_sentences.opus",
		ChannelCount:  1,
		StreamCount:   1,
		SampleRate:    48000,
		Duration:      8272,
		SampleCount:   397053,
		FileSize:      65999,
		BitRate:       63829,
		BitsPerSample: 0,
		Codec:         "opus",
		Format:        "ogg",
		//CodecLong:     "Opus", // seems to vary on different systems: Opus, Opus (Opus Interactive Audio Codec)
		FormatLong: "Ogg",
	}

	got.CodecLong = ""
	if exp != got {
		t.Errorf("Expected\n%#v\n got\n%#v", exp, got)
	}
}

func TestInfoExtractorFlac(t *testing.T) {
	ie, err := NewInfoExtractor()
	if err != nil {
		t.Errorf("got error from NewInfoExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.flac")
	got, err := ie.Process(fName)
	if err != nil {
		t.Errorf("got error from InfoExtractor.Process: %v", err)
		return
	}

	exp := AudioInfo{
		FileName:      "../test_data/three_sentences.flac",
		ChannelCount:  1,
		StreamCount:   1,
		SampleRate:    44100,
		Duration:      8265,
		SampleCount:   364505,
		FileSize:      221725,
		BitRate:       214604,
		BitsPerSample: 0,
		Codec:         "flac",
		CodecLong:     "FLAC (Free Lossless Audio Codec)",
		Format:        "flac",
		FormatLong:    "raw FLAC",
	}

	if exp != got {
		t.Errorf("Expected\n%#v\n got\n%#v", exp, got)
	}
}
