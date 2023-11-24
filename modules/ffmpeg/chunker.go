package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stts-se/transtool-open/protocol"
)

// Chunker is used to split audio files on silence, creating a subset of "phrases" from the file.
// For initialization, use NewChunker().
type Chunker struct {
	MinSilenceLen int64 // milliseconds
	ExtendChunk   int64 // milliseconds
}

// NewChunker creates a new Chunker after first checking that the ffmpeg command exists
func NewChunker(minSilenceLen, extendChunk int64) (Chunker, error) {
	if err := ffmpegEnabled(); err != nil {
		return Chunker{}, err
	}
	return Chunker{MinSilenceLen: minSilenceLen, ExtendChunk: extendChunk}, nil
}

// NewDefaultChunker creates a new Chunker after first checking that the ffmpeg command exists
func NewDefaultChunker() (Chunker, error) {
	if err := ffmpegEnabled(); err != nil {
		return Chunker{}, err
	}
	return Chunker{MinSilenceLen: DefaultMinSilenceLen, ExtendChunk: DefaultExtendChunk}, nil
}

var (
	silenceStartRE = regexp.MustCompile(".*] silence_start: ([0-9.]+) *")
	silenceEndRE   = regexp.MustCompile(".*] silence_end: ([0-9.]+) *")
	durationRE     = regexp.MustCompile("Duration: ([0-9]+):([0-9]{2}):([0-9]{2}[.][0-9]+)")
)

const (
	DefaultExtendChunk   int64 = 250 // extend all chunks by N ms before and after (N*2 ms in total)
	DefaultMinSilenceLen int64 = 1000
)

// ProcessChunk the audioFile into time chunks
func (ch Chunker) ProcessChunk(audioFile string, chunk protocol.Chunk) ([]protocol.Chunk, error) {
	res := []protocol.Chunk{}
	//log.Printf("chunker input chunk: %#v", chunk)
	tmpRes, err := ch.ProcessFile(audioFile)
	if err != nil {
		return res, err
	}
	for _, ch := range tmpRes {
		if ch.Start >= chunk.Start && ch.End <= chunk.End {
			res = append(res, ch)
		} else if ch.Start >= chunk.Start && ch.Start <= chunk.End {
			chx := protocol.Chunk{Start: ch.Start, End: chunk.End}
			//log.Printf("chunker [warning] cropping inner chunk %#v => %#v to fit into chunk %#v", ch, chx, chunk)
			res = append(res, chx)
		} else if ch.End <= chunk.End && ch.End >= chunk.Start {
			chx := protocol.Chunk{Start: chunk.Start, End: ch.End}
			//log.Printf("chunker [warning] cropping inner chunk %#v => %#v to fit into chunk %#v", ch, chx, chunk)
			res = append(res, chx)
		}
	}
	//log.Printf("Chunks %#v", res)
	return res, nil
}

// ProcessFile the audioFile into time chunks
func (ch Chunker) ProcessFile(audioFile string) ([]protocol.Chunk, error) {
	res := []protocol.Chunk{}

	minSilenceLen := float64(ch.MinSilenceLen) / 1000.0

	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return res, fmt.Errorf("no such file: %s", audioFile)
	}
	//ffmpeg -i <LJUDFIL> -af silencedetect=noise=-50dB:d=1 -f null -
	cmd := exec.Command(FfmpegCmd, "-i", audioFile, "-af", fmt.Sprintf("silencedetect=noise=-50dB:d=%.3f", minSilenceLen), "-f", "null", "-")
	//log.Printf("chunker cmd: %v", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return res, fmt.Errorf("command %s failed : %#v", cmd, err)
	}

	var totalDuration int64

	currInterval := protocol.Chunk{Start: 0}
	for _, l := range strings.Split(string(out), "\n") {
		durM := durationRE.FindStringSubmatch(l)
		if len(durM) > 0 {
			//log.Println("dur", durM[0])
			h := durM[1]
			m := durM[2]
			s := durM[3]
			secFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			ms := int64(secFloat * 1000)
			fmtS := fmt.Sprintf("%sh%sm0s", h, m)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			dur, err := time.ParseDuration(fmtS)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			totalDuration = dur.Milliseconds() + ms
			//log.Println("totalDuration", totalDuration)
		}

		startM := silenceStartRE.FindStringSubmatch(l)
		if len(startM) > 0 {
			//log.Println("start", startM[0])
			s := startM[1]
			timePointFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			timePoint0 := int64(timePointFloat * 1000)
			timePoint := timePoint0
			if timePoint0 < totalDuration+ch.ExtendChunk {
				timePoint = timePoint0 + ch.ExtendChunk
			}
			currInterval.End = timePoint
			if currInterval.Start != 0 || timePoint0 != 0 {
				res = append(res, currInterval)
				currInterval = protocol.Chunk{}
			}
		}
		endM := silenceEndRE.FindStringSubmatch(l)
		if len(endM) > 0 {
			//log.Println("end", endM[0])
			s := endM[1]
			timePointFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			timePoint := int64(timePointFloat * 1000)
			if timePoint > ch.ExtendChunk {
				timePoint = timePoint - ch.ExtendChunk
			}
			currInterval.Start = timePoint
		}
	}
	if currInterval.End == 0 && currInterval.Start != 0 {
		currInterval.End = totalDuration
		res = append(res, currInterval)
	}
	return res, nil
}
