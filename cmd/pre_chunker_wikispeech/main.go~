package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/modules"
	"github.com/stts-se/transtool/modules/ffprobe"
	"github.com/stts-se/transtool/protocol"
)

func appendPluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func baseName(fName string) string {
	res := path.Base(fName)
	var ext = filepath.Ext(fName)
	res = res[0 : len(res)-len(ext)]
	return res
}

func createUUID() string {
	uuid, err := uuid.NewUUID()
	if err != nil {
		log.Fatal("Couldn't create uuid : %v", err)
	}
	return fmt.Sprintf("%v", uuid)
}

func createPageID(audioFile string, index int) string {
	f := strings.Replace(path.Base(audioFile), ".", "_", -1)
	return fmt.Sprintf("%s_%04d", f, (index + 1))
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func potentialPageBreakOLD(accPageChunks []protocol.Chunk, nextChunk protocol.Chunk) bool {
	if *onePagePerFile {
		return false
	}
	if len(accPageChunks) == 0 {
		return false
	}
	pauseLen := nextChunk.Start - accPageChunks[len(accPageChunks)-1].End
	pageLen := accPageChunks[len(accPageChunks)-1].End - accPageChunks[0].Start

	log.Info("potentialPageBreak: pauseLen = %d, pageLen = %d", pauseLen, pageLen)

	
	if *maxChunksPerPage > 0 && int64(len(accPageChunks)) >= *maxChunksPerPage {
		log.Info("maxChunksPerPage exceeded, returning True (???)")
		return true
	}
	if *minPagePause > 0 && pauseLen < *minPagePause {
		log.Info("pauseLen less than minPagePause, returning false")
		return false
	}
	if *minPageLen > 0 && pageLen < *minPageLen {
		log.Info("pageLen less than minPageLen, returning false")
		return false
	}
	// if *minPagePause > 0 && pauseLen >= *minPagePause {
	// 	return true
	// }
	if *maxPageLen > 0 && pageLen >= *maxPageLen {
		log.Info("pageLen more than maxPageLen, returning true (???)")
		return true
	}
	//HB 220119 adding this
	if *minPageLen > 0 && pageLen >= *minPageLen && *minPagePause > 0 && pauseLen >= *minPagePause {
		log.Info("pageLen more than minPageLen and pauseLen more than minPagePause, returning true")
		return true
	}
	log.Info("No condition met, returning false")
	return false
}


//HB 220119 changed name of function - it isn't "potential", it decides if it's a pageBreak or not
//Maybe it _should_ be "potential" and do another check to see if the pages make sense? Not right now anyway..
func pageBreak(accPageChunks []protocol.Chunk, nextChunk protocol.Chunk) bool {
	if *onePagePerFile {
		return false
	}
	if len(accPageChunks) == 0 {
		return false
	}
	pauseLen := nextChunk.Start - accPageChunks[len(accPageChunks)-1].End
	pageLen := accPageChunks[len(accPageChunks)-1].End - accPageChunks[0].Start

	log.Info("potentialPageBreak: pauseLen = %d, pageLen = %d", pauseLen, pageLen)

	
	//HB 220119 adding this and changed order of conditions and removed retrun false conditions
	if *minPageLen > 0 && pageLen >= *minPageLen && *minPagePause > 0 && pauseLen >= *minPagePause {
		log.Info("pageLen more than minPageLen and pauseLen more than minPagePause, returning true")
		return true
	}

	if *maxPageLen > 0 && pageLen >= *maxPageLen {
		log.Info("pageLen more than maxPageLen, returning true")
		return true
	}

	if *maxChunksPerPage > 0 && int64(len(accPageChunks)) >= *maxChunksPerPage {
		log.Info("maxChunksPerPage exceeded, returning True (???)")
		return true
	}

	//log.Info("No condition met, returning false")
	return false
}


func audioFile2Annotations(wsAnnotator modules.WikispeechAnnotator, aiExtractor ffprobe.InfoExtractor, audioFile string) ([]protocol.AnnotationPayload, error) {
	res := []protocol.AnnotationPayload{}
	chunks, err := wsAnnotator.VAD(audioFile)
	if err != nil {
		return res, fmt.Errorf("got error from vad chunker for input file %s : %v", audioFile, err)
	}

	if len(chunks) == 0 {
		return res, fmt.Errorf("no output pages for input file %s", audioFile)
	}
	var lastAnno = func() protocol.AnnotationPayload {
		if len(res) > 0 {
			return res[len(res)-1]
		}
		return protocol.AnnotationPayload{}
	}

	var accPages [][]protocol.Chunk
	var accChunks []protocol.Chunk
	for i, chunk := range chunks {
		//HB 220119 changed name of function - it isn't "potential", it decides if it's a pageBreak or not
		//Maybe it _should_ be "potential" and do another check to see if the pages make sense? Not right now anyway..
		//if potentialPageBreak(accChunks, chunk) {
		//	log.Info("potentialPageBreak = TRUE, chunk %v", i)
		if pageBreak(accChunks, chunk) {
			log.Info("pageBreak = TRUE, chunk %v", i)
			accPages = append(accPages, accChunks)
			accChunks = []protocol.Chunk{}
		}
		accChunks = append(accChunks, chunk)
	}
	if len(accChunks) > 0 {
		accPages = append(accPages, accChunks)
	}
	log.Info("Number of pages created = %v", len(accPages))

	for i, chunks := range accPages {
		lastChunk := chunks[len(chunks)-1]

		pid := createPageID(audioFile, i)
		anno := protocol.AnnotationPayload{
			Page: protocol.PagePayload{
				ID:    pid,
				Audio: path.Base(audioFile),
			},
			//HB 0729 Adding page_status to annotation file
			CurrentStatus: protocol.Status{Name: "normal", Source: "pre_chunker", Timestamp: timestamp()},
		}
		anno.Page.Start = chunks[0].Start
		la := lastAnno()
		if la.Page.End > 0 {
			// no gaps between pages
			gap := anno.Page.Start - la.Page.End
			half := int64(float64(gap) * 0.5)
			la.Page.End += half
			res[len(res)-1] = la
			anno.Page.Start = la.Page.End
		} else if i == 0 {
			anno.Page.Start = 0
		}
		if i == len(accPages)-1 {
			info, err := aiExtractor.Process(audioFile)
			if err != nil {
				return res, fmt.Errorf("got error from audio-info for input file %s : %v", audioFile, err)
			}
			anno.Page.End = info.Duration
		} else {
			anno.Page.End = lastChunk.End
		}

		for _, chunk := range chunks {
			tc := protocol.TransChunk{
				UUID:          createUUID(),
				CurrentStatus: protocol.Status{Name: "unchecked", Source: "pre_chunker", Timestamp: timestamp()},
			}
			tc.Start = chunk.Start
			tc.End = chunk.End
			anno.Chunks = append(anno.Chunks, tc)
		}

		log.Info("Created annotation %s %d-%d (%dms)", anno.Page.ID, anno.Page.Start, anno.Page.End, anno.Page.End-anno.Page.Start)
		res = append(res, anno)
	}

	for i := 1; i < len(res); i++ {
		lastAnno := res[i-1]
		anno := res[i]
		if anno.Page.Start >= anno.Page.End {
			return res, fmt.Errorf("page end must be after page start, found start: %v, end: %v", anno.Page.Start, anno.Page.End)
		}
		if lastAnno.Page.End != anno.Page.Start {
			return res, fmt.Errorf("there's a gap between pages %v and %v", lastAnno.Page, anno.Page)
		}
	}

	return res, nil
}

func fileTotLen(anno []protocol.AnnotationPayload) int64 {
	var res int64
	for _, a := range anno {
		t := a.Page.End - a.Page.Start
		res += t
	}
	return res
}

func annotationChunksTotLen(anno []protocol.AnnotationPayload) int64 {
	var res int64
	for _, a := range anno {
		for _, c := range a.Chunks {
			t := c.End - c.Start
			res += t
		}
	}
	return res
}

// OPTIONS
var (
	minPageLen, maxPageLen, minPagePause, maxChunksPerPage *int64
	//minChunkLen, maxChunkLen, minChunkPause *int64
	project            *string
	sourceDir, annoDir string
	onePagePerFile     *bool
	//noWarnings         *bool
)

func main() {

	cmd := "pre_chunker_wikispeech"

	project = flag.String("project", "", "Project `folder`")
	minPagePause = flag.Int64("minpagepause", 0, "Minimum pause between pages (milliseconds)")
	minPageLen = flag.Int64("minpagelen", 0, "Minimum length for pages (milliseconds)")
	maxPageLen = flag.Int64("maxpagelen", 0, "Maximum length for pages (milliseconds)")
	maxChunksPerPage = flag.Int64("maxchunksperpage", 0, "Maximum number of chunks per page")
	wikispeechAnnotatorDir := flag.String("wikispeech_annotator", "", "Wikispeech annotator `directory`")

	onePagePerFile = flag.Bool("onepageperfile", false, "Pre-paginated input audio files")

	// NL 20211007 Filter out long/short files
	skipAudioWithLengthOverMs := flag.Int64("skip_audio_with_length_over_ms", -1, "Skip audio files that are longer than this number of milliseconds")
	skipAudioWithLengthUnderMs := flag.Int64("skip_audio_with_length_under_ms", -1, "Skip audio files that are shorter than this number of milliseconds")

	// NL 20211007 Filter out files with large/small portion of chunks
	skipAudioWithChunkRatioOverPercent := flag.Int64("skip_audio_with_chunk_ratio_over_percent", -1, "Skip audio files that has total time of chunks compared to total time of file over this percentage")
	skipAudioWithChunkRatioUnderPercent := flag.Int64("skip_audio_with_chunk_ratio_under_percent", -1, "Skip audio files that has total time of chunks compared to total time of file under this percentage")

	help := flag.Bool("help", false, "Print usage and exit")
	h := flag.Bool("h", false, "Print usage and exit")
	//noWarnings = flag.Bool("nowarnings", false, "Suppress warnings")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CLI to extract silence separated phrases from input audio/json source files, and generate json source/annotation files for transtool\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <options> <audio/json source file>\n", cmd)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	if *help || *h {
		flag.Usage()
		os.Exit(0)
	}

	if *project == "" {
		fmt.Fprintf(os.Stderr, "invalid input: Required flag project is not set: project\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "invalid input: Input file (audio/json) not provided (should be specified after flags)\n")
		flag.Usage()
		os.Exit(1)
	}

	if *wikispeechAnnotatorDir != "" {
		modules.WikispeechAnnotatorDir = *wikispeechAnnotatorDir
	}

	if _, err := os.Stat(*project); os.IsNotExist(err) {
		log.Fatal("Project folder %s does not exist", *project)
	}

	sourceDir = fmt.Sprintf("%s/source", strings.TrimSuffix(*project, "/"))
	annoDir = fmt.Sprintf("%s/annotation", strings.TrimSuffix(*project, "/"))

	fmt.Fprintf(os.Stderr, "Project: %s\n", *project)
	fmt.Fprintf(os.Stderr, "Source directory: %s\n", sourceDir)
	fmt.Fprintf(os.Stderr, "Annotation directory: %s\n", annoDir)

	fmt.Fprintf(os.Stderr, "Min page pause:       %7d\n", *minPagePause)
	fmt.Fprintf(os.Stderr, "Min page length:      %7d\n", *minPageLen)
	fmt.Fprintf(os.Stderr, "Max page length:      %7d\n", *maxPageLen)
	fmt.Fprintf(os.Stderr, "Max chunks per page:  %7d\n", *maxChunksPerPage)
	// fmt.Fprintf(os.Stderr, "Min chunk pause:      %7d\n", *minChunkPause)
	// fmt.Fprintf(os.Stderr, "Min chunk length:     %7d\n", *minChunkLen)
	// fmt.Fprintf(os.Stderr, "Max chunk length:     %7d\n", *maxChunkLen)

	fmt.Fprintf(os.Stderr, "One page per file:    %7v\n", *onePagePerFile)
	//fmt.Fprintf(os.Stderr, "Suppress warnings:    %7v\n", *noWarnings)
	fmt.Fprintf(os.Stderr, "\n")

	wsAnnotator, err := modules.NewWikispeechAnnotator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create ws annotator: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	aiExtractor, err := ffprobe.NewInfoExtractor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create info extractor: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	nChunks := 0
	nPages := 0
	nFiles := len(flag.Args())

	badFiles := []string{}
	skippedFiles := []string{}
	for _, audioFile := range flag.Args() {
		log.Info("Processing %s ", audioFile)
		annotations, err := audioFile2Annotations(wsAnnotator, aiExtractor, audioFile)
		if err != nil {
			//log.Fatal("Couldn't process audio file %s: %v", audioFile, err)
			//HB don't exit on error!
			log.Warning("Couldn't process audio file %s: %v", audioFile, err)
			badFiles = append(badFiles, audioFile)
			continue
		}

		// NL 20211007 ->
		totLen := fileTotLen(annotations)
		log.Info(">>> audio totLen: %d", totLen)

		if *skipAudioWithLengthOverMs > -1 && totLen > *skipAudioWithLengthOverMs {
			log.Info("Skipping file over %d ms long:\t%d\t%s", *skipAudioWithLengthOverMs, totLen, audioFile)
			skippedFiles = append(skippedFiles, audioFile)
			continue
		}
		if *skipAudioWithLengthUnderMs > -1 && totLen < *skipAudioWithLengthUnderMs {
			log.Info("Skipping file under %d ms long:\t%d\t%s", *skipAudioWithLengthUnderMs, totLen, audioFile)
			skippedFiles = append(skippedFiles, audioFile)
			continue
		}

		chunksTotLen := annotationChunksTotLen(annotations)
		chunkRatioPercent := (float64(chunksTotLen) / float64(totLen)) * 100.0

		log.Info(">>> chunksTotLen: %d", chunksTotLen)
		log.Info(">>> chunkRatioPercent: %f", chunkRatioPercent)

		if *skipAudioWithChunkRatioOverPercent > -1 && chunkRatioPercent > float64(*skipAudioWithChunkRatioOverPercent) {
			log.Info("Skipping file with chunk ration > %d %%:\t%f\t%s", *skipAudioWithChunkRatioOverPercent, chunkRatioPercent, audioFile)
			skippedFiles = append(skippedFiles, audioFile)
			continue
		}

		if *skipAudioWithChunkRatioUnderPercent < -1 && chunkRatioPercent < float64(*skipAudioWithChunkRatioUnderPercent) {
			log.Info("Skipping file with chunk ration < %d %%:\t%f\t%s", *skipAudioWithChunkRatioUnderPercent, chunkRatioPercent, audioFile)
			skippedFiles = append(skippedFiles, audioFile)
			continue
		}

		// <- NL 20211007
		var pages []protocol.PagePayload
		for i, anno := range annotations {
			pages = append(pages, anno.Page)
			nPages++
			nChunks += len(anno.Chunks)

			// force create folders on first run
			if i == 0 {
				if _, err := os.Stat(annoDir); os.IsNotExist(err) {
					os.Mkdir(annoDir, os.ModePerm)
				}
				if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
					os.Mkdir(sourceDir, os.ModePerm)
				}
			}

			json, err := json.MarshalIndent(anno, " ", " ")
			if err != nil {
				log.Fatal("Marshal failed: %v", err)
			}

			annoJSONFile := path.Join(annoDir, fmt.Sprintf("%s.json", anno.Page.ID))
			file, err := os.Create(annoJSONFile)
			if err != nil {
				log.Fatal("Couldn't create file %s : %v", annoJSONFile, err)
			}
			//HB 0810 defer file.Close()
			file.Write(json)
			file.Close()
			log.Info("Saved annotation to file %v", annoJSONFile)
		}

		json, err := json.MarshalIndent(pages, " ", " ")
		if err != nil {
			log.Fatal("Marshal failed: %v", err)
		}

		baseName := baseName(audioFile)
		sourceJSONFile := path.Join(sourceDir, fmt.Sprintf("%s.json", baseName))
		file, err := os.Create(sourceJSONFile)
		if err != nil {
			log.Fatal("Couldn't create source file %s : %v", sourceJSONFile, err)
		}
		//HB 0810defer file.Close()
		file.Write(json)
		file.Close()
		log.Info("Saved annotation to file %v", sourceJSONFile)
		//fmt.Fprintf(os.Stderr, " %s", sourceJSONFile)
	}
	log.Info("Created %d chunks, %d pages for %d audio file%s", nChunks, nPages, nFiles, appendPluralS(nFiles))

	for _, fn := range skippedFiles {
		fmt.Printf("SKIPPED FILE:\t%s\n", fn)
	}
	for _, fn := range badFiles {
		fmt.Printf("BAD FILE:\t%s\n", fn)
	}
}
