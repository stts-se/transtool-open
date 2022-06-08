package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/modules/ffmpeg"
	"github.com/stts-se/transtool/protocol"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const debug = false

// GoogleASR is used to call Google's Speech API for recognition. For initialization, use NewGoogleASR().
type GoogleASR struct {
	ctx     context.Context
	client  *speech.Client
	chunkex ffmpeg.ChunkExtractor
}

// NewGoogleASR creates a new GoogleASR after first initializing some stuff
func NewGoogleASR(credentialsFile string) (GoogleASR, error) {
	res := GoogleASR{}

	chunkex, err := ffmpeg.NewChunkExtractor()
	if err != nil {
		return res, fmt.Errorf("couldn't initialize ChunkExtractor : %v", err)
	}
	res.chunkex = chunkex

	credentialsFileAbs, err := filepath.Abs(credentialsFile)
	if err != nil {
		return res, fmt.Errorf("couldn't get absolut path to credentials file : %v", err)
	}
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsFileAbs); err != nil {
		return res, fmt.Errorf("couldn't set Google API credentials : %v", err)
	}
	log.Info("[google_asr] Google API credentials: %s", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))

	res.ctx = context.Background()

	// Create a client.
	client, err := speech.NewClient(res.ctx)
	if err != nil {
		return res, fmt.Errorf("failed to create client : %v", err)
	}
	log.Info("[google_asr] Created Google Speech API Client")
	res.client = client
	return res, nil
}

// Process runs Google ASR on each part of the file as specified in the `chunks` input. If the chunk list is empty, the whole file will be processed.
func (gASR GoogleASR) Process(config protocol.ASRConfig, audioPath string, chunk protocol.Chunk) (protocol.ASROutput, error) {
	var err error
	res := protocol.ASROutput{}

	if debug {
		fmt.Println("GoogleASR debug", audioPath, chunk)
	}

	if chunk.Start > chunk.End {
		return res, fmt.Errorf("cannot process input chunk with negative duration: %v-%v", chunk.Start, chunk.End)
	}
	if chunk.Start == chunk.End && chunk.Start > 0 {
		return res, fmt.Errorf("cannot process input chunk with zero duration: %v-%v", chunk.Start, chunk.End)
	}

	var gEnc speechpb.RecognitionConfig_AudioEncoding
	switch config.Encoding {
	case "wav":
		gEnc = speechpb.RecognitionConfig_LINEAR16
	case "flac":
		gEnc = speechpb.RecognitionConfig_FLAC
	case "opus":
		gEnc = speechpb.RecognitionConfig_OGG_OPUS
	default:
		return res, fmt.Errorf("unknown encoding: %s", config.Encoding)
	}

	gConfig := &speechpb.RecognitionConfig{
		Encoding:              gEnc,
		SampleRateHertz:       int32(config.SampleRate),
		LanguageCode:          config.Lang,
		AudioChannelCount:     int32(config.ChannelCount),
		EnableWordTimeOffsets: true,
	}

	var data []byte
	if chunk.Start == 0 && chunk.End == 0 && gEnc != speechpb.RecognitionConfig_OGG_OPUS {
		data, err = os.ReadFile(audioPath)
		if err != nil {
			return res, fmt.Errorf("failed to read file : %v", err)
		}
	} else {
		tmpData, err := gASR.chunkex.ProcessFile(audioPath, []protocol.Chunk{chunk}, "flac")
		gConfig.Encoding = speechpb.RecognitionConfig_FLAC
		gConfig.AudioChannelCount = 1 // todo: how do we know?
		if err != nil {
			return res, fmt.Errorf("failed to extract chunks : %v", err)
		}
		if len(tmpData) != 1 {
			return res, fmt.Errorf("failed to extract chunks : empty byte array")
		}
		data = tmpData[0]
		if debug {
			tmpFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("google-asr-chunk-%s-%v-%v-", filepath.Base(audioPath), chunk.Start, chunk.End))
			if err != nil {
				log.Error("[google_asr] Couldn't create temporary file: %v", err)
			} else {
				defer tmpFile.Close()
				//defer os.Remove(tmpFile)
				_, err := tmpFile.Write(data)
				if err != nil {
					log.Info("Couldn't write data to temp file: %v", err)
				}
				log.Debug("[google_asr] GoogleASR debug tempfile %s", tmpFile.Name())
			}
		}
	}

	// Detect speech in the audio file
	resp, err := gASR.client.Recognize(gASR.ctx, &speechpb.RecognizeRequest{
		Config: gConfig,
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data},
		},
	})
	if err != nil {
		return res, fmt.Errorf("failed to recognize %v bytes of data : %v", len(data), err)
	}

	// Process results
	for _, result := range resp.Results {
		if len(result.Alternatives) > 0 {
			alt := result.Alternatives[0]
			chunks := []protocol.ASROutputChunk{}
			for _, token := range alt.Words {
				start := token.StartTime.AsDuration().Milliseconds()
				end := token.EndTime.AsDuration().Milliseconds()
				//fmt.Printf("GoogleASR debug %v %v %v\n", token.Word, start, end)
				chunk := protocol.ASROutputChunk{
					Text: token.Word,
					Chunk: protocol.Chunk{
						Start: start,
						End:   end,
					},
				}
				chunks = append(chunks, chunk)
			}
			res = protocol.ASROutput{Chunks: chunks}
		} else {
			res = protocol.ASROutput{}
		}
	}
	return res, nil
}

func (gASR GoogleASR) Initialised() bool {
	return gASR.client != nil
}
