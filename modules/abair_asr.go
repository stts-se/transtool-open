package modules

import (
        "os"
	"encoding/json"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
    	b64 "encoding/base64"
	"strings"

	"context"
	"path/filepath"

	"github.com/stts-se/transtool-open/log"
	"github.com/stts-se/transtool-open/modules/ffmpeg"
	"github.com/stts-se/transtool-open/protocol"

    )

const debugx = true

// AbairASR is used to call Abair Speech API for recognition. For initialization, use NewAbairASR().
type AbairASR struct {
	ctx     context.Context
	//client  *speech.Client
	chunkex ffmpeg.ChunkExtractor
}

// NewAbairASR creates a new AbairASR after first initializing some stuff
func NewAbairASR() (AbairASR, error) {
	res := AbairASR{}

	chunkex, err := ffmpeg.NewChunkExtractor()
	if err != nil {
		return res, fmt.Errorf("couldn't initialize ChunkExtractor : %v", err)
	}
	res.chunkex = chunkex

	res.ctx = context.Background()

	return res, nil
}


func check(e error) {
    if e != nil {
        panic(e)
    }
}


type AsrRequest struct {
    RecogniseBlob string `json:"recogniseBlob"`
    Method   string `json:"method"`
    Developer bool `json:"developer"`
}

type AsrResponse struct {
	AudioFilePath string `json:"audioFilePath"`
	Transcriptions []Transcription `json:"transcriptions"`
	Duration float64 `json:"duration"`
}

type Transcription struct {
     Utterance string `json:"utterance"`
}



//func Process() {
// Process runs Abair ASR on each part of the file as specified in the `chunks` input. If the chunk list is empty, the whole file will be processed.
func (aASR AbairASR) Process(config protocol.ASRConfig, audioPath string, chunk protocol.Chunk) (protocol.ASROutput, error) {
	var err error
	res := protocol.ASROutput{}

	if debugx {
		fmt.Println("AbairASR debug", audioPath, chunk)
	}

	if chunk.Start > chunk.End {
		return res, fmt.Errorf("cannot process input chunk with negative duration: %v-%v", chunk.Start, chunk.End)
	}
	if chunk.Start == chunk.End && chunk.Start > 0 {
		return res, fmt.Errorf("cannot process input chunk with zero duration: %v-%v", chunk.Start, chunk.End)
	}

	var data []byte
	if chunk.Start == 0 && chunk.End == 0 {
		data, err = os.ReadFile(audioPath)
		if err != nil {
			return res, fmt.Errorf("failed to read file : %v", err)
		}
	} else {
		tmpData, err := aASR.chunkex.ProcessFile(audioPath, []protocol.Chunk{chunk}, "flac")
		//gConfig.Encoding = speechpb.RecognitionConfig_FLAC
		//gConfig.AudioChannelCount = 1 // todo: how do we know?
		if err != nil {
			return res, fmt.Errorf("failed to extract chunks : %v", err)
		}
		if len(tmpData) != 1 {
			return res, fmt.Errorf("failed to extract chunks : empty byte array")
		}
		data = tmpData[0]
		if debugx {
			tmpFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("abair-asr-chunk-%s-%v-%v-", filepath.Base(audioPath), chunk.Start, chunk.End))
			if err != nil {
				log.Error("[abair_asr] Couldn't create temporary file: %v", err)
			} else {
				defer tmpFile.Close()
				//defer os.Remove(tmpFile)
				_, err := tmpFile.Write(data)
				if err != nil {
					log.Info("Couldn't write data to temp file: %v", err)
				}
				log.Debug("[abair_asr] AbairASR debug tempfile %s", tmpFile.Name())
			}
		}
	}


	//HB 


	encodedAudio := b64.StdEncoding.EncodeToString([]byte(data))


	httpposturl := "https://phoneticsrv3.lcs.tcd.ie/asr_api/recognise"
	//fmt.Println("HTTP JSON POST URL:", httpposturl)

	asrRequest := AsrRequest{
    	      RecogniseBlob: encodedAudio,
    	      Method:   "online2bin",
	      Developer: true,
	}
	jsonData, err := json.Marshal(asrRequest)
	check(err)
	
	//fmt.Println("jsonData:", jsonData)
	//var f interface{}
	//err = json.Unmarshal(jsonData, &f)
	//fmt.Println("f:", f)

	request, err := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
	check(err)
	
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	check(err)

	defer response.Body.Close()

	//fmt.Println("response Status:", response.Status)
	//fmt.Println("response Headers:", response.Header)
	body, err := ioutil.ReadAll(response.Body)
	check(err)

	//fmt.Println("response Body:", string(body))
	
	asrResponse := AsrResponse{}
	jsonErr := json.Unmarshal(body, &asrResponse)
	check(jsonErr)
	//fmt.Println(asrResponse.AudioFilePath)

	var resText = strings.TrimSpace(asrResponse.Transcriptions[0].Utterance)

	fmt.Println(resText)

	//return asrResponse

	chunks := []protocol.ASROutputChunk{}
	reschunk := protocol.ASROutputChunk{
		Text: resText,
		Chunk: protocol.Chunk{
			Start: 0,
			End:   0,
			},
		}
	chunks = append(chunks, reschunk)
	res = protocol.ASROutput{Chunks: chunks}

	return res, nil



}
