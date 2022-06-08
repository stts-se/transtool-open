package ffprobe

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"

	//"log"
	"os"
)

// {
//     "streams": [
//         {
//             "index": 0,
//             "codec_name": "pcm_s16le",
//             "codec_long_name": "PCM signed 16-bit little-endian",
//             "codec_type": "audio",
//             "codec_time_base": "1/44100",
//             "codec_tag_string": "[1][0][0][0]",
//             "codec_tag": "0x0001",
//             "sample_fmt": "s16",
//             "sample_rate": "44100",
//             "channels": 1,
//             "bits_per_sample": 16,
//             "r_frame_rate": "0/0",
//             "avg_frame_rate": "0/0",
//             "time_base": "1/44100",
//             "duration_ts": 364505,
//             "duration": "8.265420",
//             "bit_rate": "705600",
//             "disposition": {
//                 "default": 0,
//                 "dub": 0,
//                 "original": 0,
//                 "comment": 0,
//                 "lyrics": 0,
//                 "karaoke": 0,
//                 "forced": 0,
//                 "hearing_impaired": 0,
//                 "visual_impaired": 0,
//                 "clean_effects": 0,
//                 "attached_pic": 0
//             }
//         }
//     ],
//     "format": {
//         "filename": "modules/test_data/three_sentences.wav",
//         "nb_streams": 1,
//         "nb_programs": 0,
//         "format_name": "wav",
//         "format_long_name": "WAV / WAVE (Waveform Audio)",
//         "duration": "8.265420",
//         "size": "729054",
//         "bit_rate": "705642",
//         "probe_score": 99
//     }
// }
type ffprobeInfo struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecName     string `json:"codec_name"`
	CodecLongName string `json:"codec_long_name"`
	SampleRate    string `json:"sample_rate"`
	Channels      int64  `json:"channels"`
	BitsPerSample int64  `json:"bits_per_sample"`
	BitRate       string `json:"bit_rate"`
	Duration      string `json:"duration"`    // seconds
	SampleCount   int64  `json:"duration_ts"` // samples
}

type ffprobeFormat struct {
	FileName       string `json:"filename"`
	Streams        int64  `json:"nb_streams"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
	// "nb_programs": 0,
	// "probe_score": 99
}

type AudioInfo struct {
	FileName     string `json:"file_name"`
	ChannelCount int64  `json:"channel_count"`
	StreamCount  int64  `json:"stream_count"`
	SampleRate   int64  `json:"sample_rate"`

	// Duration in milliseconds
	Duration int64 `json:"duration"`

	SampleCount   int64 `json:"sample_count"`
	FileSize      int64 `json:"file_size"`
	BitRate       int64 `json:"bit_rate"`
	BitsPerSample int64 `json:"bits_per_sample,omitempty"`

	Codec      string `json:"codec"`
	CodecLong  string `json:"codec_long"`
	Format     string `json:"format"`
	FormatLong string `json:"format_long"`
}

func audioInfoFromFFProbeInfo(ff ffprobeInfo) (AudioInfo, error) {
	// if len(ff.Streams) != 1 {
	// 	return AudioInfo{}, fmt.Errorf("failed to parse audio with %d streams", len(ff.Streams))
	// }
	stream := ff.Streams[0]

	durSeconds, err := strconv.ParseFloat(ff.Format.Duration, 64)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("failed to parse duration from string '%s'", stream.Duration)
	}
	durMillis := int64(math.Round(durSeconds * 1000))

	sampleRate, err := strconv.ParseInt(stream.SampleRate, 10, 64)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("failed to parse sample rate from string '%s'", stream.SampleRate)
	}

	// check that properties retreived from stream are not mismatching
	if len(ff.Streams) > 1 {
		firstStream := ff.Streams[0]
		for _, stream := range ff.Streams[1:] {
			if stream.CodecName == "png" {
				continue
			}
			if /* stream.SampleRate != "" && */ firstStream.SampleRate != stream.SampleRate {
				return AudioInfo{}, fmt.Errorf("mismatching sample rate for streams: %v vs %v", firstStream.SampleRate, stream.SampleRate)
			}
			if /* stream.Channels != 0 && */ firstStream.Channels != stream.Channels {
				return AudioInfo{}, fmt.Errorf("mismatching channels for streams: %v vs %v", firstStream.Channels, stream.Channels)
			}
			if /* stream.SampleCount != 0 && */ firstStream.SampleCount != stream.SampleCount {
				return AudioInfo{}, fmt.Errorf("mismatching sample count for streams: %v vs %v", firstStream.SampleCount, stream.SampleCount)
			}
			if /* stream.SampleCount != 0 && */ firstStream.BitsPerSample != stream.BitsPerSample {
				return AudioInfo{}, fmt.Errorf("mismatching bits per sample for streams: %v vs %v", firstStream.BitsPerSample, stream.BitsPerSample)
			}
			if /* stream.CodecName != "" && */ firstStream.CodecName != stream.CodecName {
				return AudioInfo{}, fmt.Errorf("mismatching codec name for streams: %v vs %v", firstStream.CodecName, stream.CodecName)
			}
			if /* stream.CodecLongName != "" && */ firstStream.CodecLongName != stream.CodecLongName {
				return AudioInfo{}, fmt.Errorf("mismatching codec long name for streams: %v vs %v", firstStream.CodecLongName, stream.CodecLongName)
			}
		}
	}

	bitRate, err := strconv.ParseInt(ff.Format.BitRate, 10, 64)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("failed to parse bitrate from string '%s'", ff.Format.BitRate)
	}
	fileSize, err := strconv.ParseInt(ff.Format.Size, 10, 64)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("failed to parse file size from string '%s'", ff.Format.Size)
	}
	return AudioInfo{
		FileName:      ff.Format.FileName,
		ChannelCount:  stream.Channels,
		StreamCount:   int64(len(ff.Streams)),
		SampleRate:    sampleRate,
		SampleCount:   stream.SampleCount,
		Duration:      durMillis,
		FileSize:      fileSize,
		BitRate:       bitRate,
		BitsPerSample: stream.BitsPerSample,
		Codec:         stream.CodecName,
		CodecLong:     stream.CodecLongName,
		Format:        ff.Format.FormatName,
		FormatLong:    ff.Format.FormatLongName,
	}, nil
}

// InfoExtractor extracts ffprobe style audio information from an audio file.
// For initialization, use NewInfoExtractor().
type InfoExtractor struct {
}

// NewInfoExtractor creates a new AudioInfo after first checking that the soxi command exists
func NewInfoExtractor() (InfoExtractor, error) {
	if err := ffprobeEnabled(); err != nil {
		return InfoExtractor{}, err
	}
	return InfoExtractor{}, nil
}

// Process extracts info for the specified file
func (ie InfoExtractor) Process(audioFile string) (AudioInfo, error) {
	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return AudioInfo{}, fmt.Errorf("no such file: %s", audioFile)
	}

	//ffprobe -v quiet -print_format json -show_format -show_streams <AUDIO>
	args := []string{"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", audioFile}
	cmd := exec.Command(FFProbeCmd, args...)
	//log.Printf("audioinfo cmd: %v", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return AudioInfo{}, fmt.Errorf("command %s failed : %v", cmd, err)
	}

	var ff ffprobeInfo
	err = json.Unmarshal(out, &ff)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("unmarshal %s failed : %v", cmd, err)
	}
	res, err := audioInfoFromFFProbeInfo(ff)
	if err != nil {
		return AudioInfo{}, fmt.Errorf("failed to parse ffprobeInfo into AudioInfo : %v", err)
	}
	return res, nil
}
