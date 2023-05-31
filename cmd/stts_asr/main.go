package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	//"sync"

	"github.com/stts-se/transtool/dbapi"
	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/modules"
	"github.com/stts-se/transtool/modules/ffprobe"
	"github.com/stts-se/transtool/protocol"
	"github.com/stts-se/transtool/validation"
)

var proj *dbapi.Proj

var sttsASR modules.SttsASR
var aiExtractor ffprobe.InfoExtractor

func runASR(asrConfig protocol.ASRConfig, audioPath string, ch protocol.TransChunk) string {
	output, err := sttsASR.Process(asrConfig, audioPath, ch.Chunk)
	if err != nil {
		msg := fmt.Sprintf("aiExtractor.Process error: %v", err)
		log.Fatal(msg)
	}
	trans := []string{}
	for _, t := range output.Chunks {
		trans = append(trans, t.Text)
	}
	return strings.Join(trans, " ")
}

func main() {

	cmd := path.Base(os.Args[0])

	projectDirs := flag.String("project_dirs", "", "Project directories separated by ':' (path1/dir1:path1/dir2 [...])")
	asrURL := flag.String("asr_url", "http://localhost:8887/recognise", "ASR `URL`")
	force := flag.Bool("force", false, "overwrite existing transcriptions")

	help := flag.Bool("help", false, "Print usage and exit")
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <flags>\n", cmd)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if strings.HasPrefix(*projectDirs, "-") {
		fmt.Fprintf(os.Stderr, "Invalid project dirs: %s\n", *projectDirs)
		flag.Usage()
		os.Exit(1)
	}
	if *projectDirs == "" {
		fmt.Fprintf(os.Stderr, "Required flag project_dirs not set\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Didn't expect cmd line args except for flags, found: %#v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	proj0, err := dbapi.NewProj(*projectDirs, &validation.Validator{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load project dir : %v", err)
		os.Exit(1)
	}
	proj = &proj0
	aiExtractor, err = ffprobe.NewInfoExtractor()
	if err != nil {
		log.Fatal("Failed to initialise AI extractor: %v", err)
	}

	sttsASR, err = modules.NewSttsASR()
	if err != nil {
		log.Warning("Failed to initialise Stts ASR: %v", err)
	}
	_, err = proj.LoadData()
	if err != nil {
		log.Fatal("Couldn't load data: %v", err)
	}

	//var wg sync.WaitGroup

	for _, pr := range proj.ListSubProjs() {
		db := proj.GetDB(pr)
		for _, anno := range db.GetAnnotationData() {
			audioPath, err := proj.BuildAudioPath(pr, anno.Page.Audio)
			if err != nil {
				msg := fmt.Sprintf("db.BuildAudioPath error: %v", err)
				log.Fatal(msg)
			}
			info, err := aiExtractor.Process(audioPath)
			if err != nil {
				msg := fmt.Sprintf("aiExtractor.Process error: %v", err)
				log.Fatal(msg)
			}
			asrConfig := protocol.ASRConfig{
				URL:      *asrURL,
				Encoding: strings.TrimPrefix(filepath.Ext(audioPath), "."),
				// Lang:         *lang,
				SampleRate:   int(info.SampleRate),   // 48000,
				ChannelCount: int(info.ChannelCount), //2,
			}
			for i, ch := range anno.Chunks {
				isUnchecked := ch.CurrentStatus.Name == "unchecked" || ch.CurrentStatus.Name == ""
				if ch.Trans != "" && !*force {
					log.Info("Skipping chunk with existing trans: %#v", ch)
					continue
				}
				if !isUnchecked && !*force {
					log.Info("Skipping chunk with status: %#v", ch)
					continue
				}
				chunkIndex := i
				annox := anno
				//wg.Add(1)
				//go func(annox *protocol.AnnotationPayload, chunkIndex int) {
				//	defer wg.Done()
				log.Info("sending chunk to asr: #%v/%v in %v", chunkIndex+1, len(annox.Chunks), annox.Page.ID)
				trans := runASR(asrConfig, audioPath, ch)
				ch.Trans = trans
				annox.Chunks[chunkIndex] = ch
				log.Info("completed: %v %v %v", annox.Page.ID, chunkIndex, ch.Trans)
				//}(&anno, i)

			}
			err = db.Save(anno)
			if err != nil {
				msg := fmt.Sprintf("dbapi.Save error: %v", err)
				log.Fatal(msg)
			}
		}
	}
	//wg.Wait()

}
