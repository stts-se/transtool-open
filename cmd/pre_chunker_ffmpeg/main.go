package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/modules/ffmpeg"
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

// secret init function
func init() {
	log.Info("Running init tests...")
	// test getSplitPoints!
	var start, maxLen, minLen int64
	var chunk protocol.Chunk
	var res, expect []protocol.Chunk
	var gap = int64(0)

	//
	maxLen = 100
	minLen = 10
	gap = 0
	chunk = protocol.Chunk{Start: 0, End: 379}
	start = chunk.Start
	res = getSplitPoints(chunk, gap, minLen, maxLen)
	expect = []protocol.Chunk{
		{Start: start, End: start + maxLen},
		{Start: start + (maxLen+gap)*1, End: start + maxLen*2 + gap*1},
		{Start: start + (maxLen+gap)*2, End: 289},
		{Start: 289, End: start + chunk.End},
	}
	if !reflect.DeepEqual(res, expect) {
		log.Fatal("init test failed: for %#v with max len %v:\nexp %#v\ngot %#v", chunk, maxLen, expect, res)
	}

	//
	maxLen = 100
	minLen = 10
	gap = 0
	chunk = protocol.Chunk{Start: start, End: start + 513}
	start = chunk.Start
	res = getSplitPoints(chunk, gap, minLen, maxLen)
	expect = []protocol.Chunk{
		{Start: start + maxLen*0, End: start + maxLen},
		{Start: start + (maxLen+gap)*1, End: start + maxLen*2 + gap*1},
		{Start: start + (maxLen+gap)*2, End: start + maxLen*3 + gap*2},
		{Start: start + (maxLen+gap)*3, End: start + maxLen*4 + gap*3},
		{Start: start + (maxLen+gap)*4, End: 456},
		{Start: 456, End: start + chunk.End},
	}
	if !reflect.DeepEqual(res, expect) {
		log.Fatal("init test failed: for %#v with max len %v, gap %v:\nexp %#v\ngot %#v", chunk, maxLen, gap, expect, res)
	}

	//
	maxLen = 100
	minLen = 10
	gap = 10
	chunk = protocol.Chunk{Start: 0, End: 379}
	start = chunk.Start
	res = getSplitPoints(chunk, gap, minLen, maxLen)
	expect = []protocol.Chunk{
		{Start: start, End: start + maxLen},
		{Start: start + (maxLen+gap)*1, End: start + maxLen*2 + gap*1},
		{Start: start + (maxLen+gap)*2, End: start + 299},
		{Start: start + 309, End: start + chunk.End},
	}
	if !reflect.DeepEqual(res, expect) {
		log.Fatal("init test failed: for %#v with max len %v, gap %v:\nexp %#v\ngot %#v", chunk, maxLen, gap, expect, res)
	}

	//
	maxLen = 100
	minLen = 10
	gap = 10
	chunk = protocol.Chunk{Start: start, End: start + 513}
	start = chunk.Start
	res = getSplitPoints(chunk, gap, minLen, maxLen)
	expect = []protocol.Chunk{
		{Start: start + maxLen*0, End: start + maxLen},
		{Start: start + (maxLen+gap)*1, End: start + maxLen*2 + gap*1},
		{Start: start + (maxLen+gap)*2, End: start + maxLen*3 + gap*2},
		{Start: start + (maxLen+gap)*3, End: start + 421},
		{Start: start + 431, End: start + chunk.End},
	}
	if !reflect.DeepEqual(res, expect) {
		log.Fatal("init test failed: for %#v with max len %v, gap %v:\nexp %#v\ngot %#v", chunk, maxLen, gap, expect, res)
	}

	log.Info("Init tests completed, and success is a fact!")
}

func getSplitPoints(chunk protocol.Chunk, splitGap, minLen, maxLen int64) []protocol.Chunk {
	if splitGap > maxLen {
		log.Fatal("getSplitPoints: Cannot split if splitGap is larger than maxLen (%d vs %d)", splitGap, maxLen)
	}
	if minLen > maxLen {
		log.Fatal("getSplitPoints: Cannot split if minLen is larger than maxLen (%d vs %d)", minLen, maxLen)
	}

	res := []protocol.Chunk{}
	currStartingPoint := chunk.Start
	rest := chunk.End - currStartingPoint
	n := 0
	for rest > 0 {
		n++
		if n > 10 {
			break
		}
		var end int64
		if rest < splitGap+maxLen*2 {
			end = currStartingPoint + (rest / 2)
			newChunk := protocol.Chunk{Start: currStartingPoint, End: end}
			res = append(res, newChunk)

			currStartingPoint = newChunk.End + splitGap
			//rest = chunk.End - currStartingPoint

			end = chunk.End
			newChunk = protocol.Chunk{Start: currStartingPoint, End: end}
			res = append(res, newChunk)
			break
		} else if rest >= maxLen {
			end = currStartingPoint + maxLen
		} else {
			end = chunk.End
		}
		newChunk := protocol.Chunk{Start: currStartingPoint, End: end}
		res = append(res, newChunk)

		currStartingPoint = newChunk.End + splitGap
		rest = chunk.End - currStartingPoint
	}

	return res
}

func audioFile2OnePage(aiExtractor ffprobe.InfoExtractor, audioFile string) (protocol.PagePayload, error) {
	info, err := aiExtractor.Process(audioFile)
	if err != nil {
		return protocol.PagePayload{}, fmt.Errorf("got error from audio-info for input file %s : %v", audioFile, err)
	}
	id := baseName(audioFile)
	res := protocol.PagePayload{
		ID:    id,
		Audio: path.Base(audioFile),
	}
	res.Start = int64(0)
	res.End = info.Duration
	return res, nil
}

func audioFile2Pages(audioFile string, pageChunker ffmpeg.Chunker) ([]protocol.PagePayload, error) {
	pChunks, err := pageChunker.ProcessFile(audioFile)
	pages := []protocol.PagePayload{}
	if err != nil {
		return pages, fmt.Errorf("got error from page chunker for input file %s : %v", audioFile, err)
	}
	if len(pChunks) == 0 {
		return pages, fmt.Errorf("no output pages for input file %s", audioFile)
	}
	shrunkChunks := []protocol.Chunk{}
	if *shrinkPages {
		for _, pc := range pChunks {
			pageLen := pc.End - pc.Start
			if pageLen > *maxOuterLen {
				pcs := shrink(pageChunker, audioFile, pc, *maxOuterLen)
				if len(pcs) > 1 {
					if !*noWarnings {
						log.Warning("Shrunk page %#v => %#v", pc, pcs)
					}
					shrunkChunks = append(shrunkChunks, pcs...)
				} else {
					shrunkChunks = append(shrunkChunks, pc)
				}
			}
		}
		pChunks = shrunkChunks
	}

	var lastPage = func() protocol.PagePayload {
		var lp protocol.PagePayload
		if len(pages) > 0 {
			return pages[len(pages)-1]
		}
		return lp
	}

	for _, pChunk := range pChunks {
		pIndex := len(pages)

		lp := lastPage()
		if lp.End > 0 {
			pChunk.Start = lp.End // no gaps between pages
		}

		pageLen := pChunk.End - pChunk.Start

		// 1) page too short, merge with last page
		if pageLen < *minOuterLen && pIndex > 0 && len(pages) > 0 && lp.End > 0 {
			if !*noWarnings {
				log.Warning("Page too short: %vms, expected min %vms, will merge with previous page -- %#v", pageLen, *minOuterLen, pChunk)
			}
			lp.End = pChunk.End
			pages[len(pages)-1] = lp
			continue
		}
		// 2) page too long, force split into parts
		if pageLen > *maxOuterLen {
			if !*noWarnings {
				log.Warning("Page too long: %vms, expected max %vms", pageLen, *maxOuterLen)
			}
			if *forceSplitPages {
				splitPoints := getSplitPoints(pChunk, outerSplitGap, *minOuterLen, *maxOuterLen)
				if !*noWarnings {
					log.Warning("Will split page into %d parts\n %#v => %#v", len(splitPoints), pChunk, splitPoints)
				}

				//fmt.Printf("CHUNK: %#v", pChunk)
				for _, split := range splitPoints {
					pIndex := len(pages)
					pid := createPageID(audioFile, pIndex)
					page := protocol.PagePayload{
						ID:    pid,
						Audio: path.Base(audioFile),
					}
					page.Start = split.Start
					page.End = split.End
					pages = append(pages, page)
					//fmt.Printf("=> PAGE: %#v", page)
				}
			}
			continue
		}

		// 3) else, all is well
		pid := createPageID(audioFile, pIndex)
		page := protocol.PagePayload{
			ID:    pid,
			Audio: path.Base(audioFile),
		}
		page.Start = pChunk.Start
		page.End = pChunk.End
		pages = append(pages, page)
	}

	for _, page := range pages {
		if page.Start >= page.End {
			return pages, fmt.Errorf("page end must be after page start, found start: %v, end: %v", page.Start, page.End)
		}
	}

	return pages, nil
}

func page2Annotation(audioFile string, page protocol.PagePayload, innerChunker ffmpeg.Chunker) (protocol.AnnotationPayload, error) {
	anno := protocol.AnnotationPayload{
		Page: page,
		// CurrentStatus: protocol.Status{Name: "unchecked", Source: "auto", Timestamp: timestamp()},
	}
	chunks, err := innerChunker.ProcessChunk(audioFile, page.Chunk)
	if err != nil {
		return anno, fmt.Errorf("got error from inner chunker for page id %s : %v", page.ID, err)
	}

	// start pre-processing
	if insertMissingChunks {
		if len(chunks) == 0 {
			if !*noWarnings {
				log.Warning("No inner chunks for page id %s, will auto-create one chunk", page.ID)
			}
			chunks = []protocol.Chunk{{Start: page.Start, End: page.End}}
		}

		// insert missing chunk at the beginning
		lcFirst := chunks[0]
		if lcFirst.Start-page.Start > *missingChunkLen {
			chunk := protocol.Chunk{
				Start: page.Start + *innerSplitGap,
				End:   lcFirst.Start - *innerSplitGap,
			}
			chunks = append([]protocol.Chunk{chunk}, chunks...)
			if !*noWarnings {
				log.Warning("Inserting extra chunk at the beginning of page %v (empty head was %v ms)", page.ID, lcFirst.Start-page.Start)
			}
		}
		// insert missing chunk at the end
		lcLast := chunks[len(chunks)-1]
		if page.End-lcLast.End > *missingChunkLen {
			chunk := protocol.Chunk{
				Start: lcLast.End + *innerSplitGap,
				End:   page.End - *innerSplitGap,
			}
			chunks = append(chunks, chunk)
			if !*noWarnings {
				log.Warning("Inserting extra chunk at the end of page %v (empty tail was %v ms)", page.ID, page.End-lcLast.End)
			}
		}
	}
	if *shrinkChunks {
		shrunkChunks := []protocol.Chunk{}
		for _, ch := range chunks {
			chLen := ch.End - ch.Start
			if chLen > *maxInnerLen {
				chs := shrink(innerChunker, audioFile, ch, *maxInnerLen)
				if len(chs) > 1 {
					if !*noWarnings {
						log.Warning("Shrunk chunk %#v => %#v", ch, chs)
					}
					shrunkChunks = append(shrunkChunks, chs...)
				} else {
					shrunkChunks = append(shrunkChunks, ch)
				}
			}
		}
		chunks = shrunkChunks
	}
	// end pre-processing

	var lastChunk = func() protocol.TransChunk {
		var lc protocol.TransChunk
		if len(anno.Chunks) > 0 {
			return anno.Chunks[len(anno.Chunks)-1]
		}
		return lc
	}

	for _, chunk := range chunks {
		lc := lastChunk()

		chunkLen := chunk.End - chunk.Start

		// empty chunk, skip and continue
		if chunkLen == 0 {
			continue
		}

		// 1) chunk too short, merge with last chunk
		if chunkLen < *minInnerLen && len(anno.Chunks) > 0 && lc.End > 0 {
			if !*noWarnings {
				log.Warning("Chunk too short: %vms, expected min %vms, will merge with previous chunk -- %#v", chunkLen, *minInnerLen, chunk)
			}
			lc.End = chunk.End
			lc.CurrentStatus = protocol.Status{Name: "unchecked", Source: "merged", Timestamp: timestamp()}
			anno.Chunks[len(anno.Chunks)-1] = lc
			continue
		}
		// 2) chunk too long, split into parts
		if chunkLen > *maxInnerLen {
			if !*noWarnings {
				log.Warning("Chunk too long: %vms, expected max %vms", chunkLen, *maxInnerLen)
			}
			if *forceSplitChunks {
				splitPoints := getSplitPoints(chunk, *innerSplitGap, *minInnerLen, *maxInnerLen)
				if !*noWarnings {
					log.Warning("Will split chunk into %d parts\n %#v => %#v", len(splitPoints), splitPoints, chunk)
				}
				//fmt.Printf("CHUNK: %#v", pChunk)
				for _, split := range splitPoints {
					tc := protocol.TransChunk{
						UUID:          createUUID(),
						CurrentStatus: protocol.Status{Name: "unchecked", Source: "split", Timestamp: timestamp()},
					}
					tc.Start = split.Start
					tc.End = split.End
					anno.Chunks = append(anno.Chunks, tc)
					//fmt.Printf("=> CHUNK: %#v", tc)
				}
			}
			continue
		}

		// 3) else, all is well
		tc := protocol.TransChunk{
			UUID:          createUUID(),
			CurrentStatus: protocol.Status{Name: "unchecked", Source: "auto", Timestamp: timestamp()},
		}
		tc.Start = chunk.Start
		tc.End = chunk.End
		anno.Chunks = append(anno.Chunks, tc)
	}

	for _, chunk := range anno.Chunks {
		if chunk.Start >= chunk.End {
			annoBts, _ := json.MarshalIndent(anno, " ", " ")
			return anno, fmt.Errorf("chunk end must be after chunk start, found start: %v, end: %v for chunk id %s\nANNO: %s", chunk.Start, chunk.End, chunk.UUID, string(annoBts))
		}
	}

	if anno.Page.Start >= anno.Page.End {
		return anno, fmt.Errorf("anno end must be after anno start, found start: %v, end: %v", anno.Page.Start, anno.Page.End)
	}
	return anno, nil
}

func innerShrink(audioFile string, lastChunker ffmpeg.Chunker, innerChunk protocol.Chunk, absMin, maxLen int64) []protocol.Chunk {
	res := []protocol.Chunk{}
	localChunker := lastChunker
	localChunker.MinSilenceLen = int64(math.Floor(float64(lastChunker.MinSilenceLen) * (.8)))
	log.Info("shrinkChunker.MinSilenceLen %v, absMin %v", localChunker.MinSilenceLen, absMin)
	if localChunker.MinSilenceLen < absMin {
		return []protocol.Chunk{innerChunk}
	}
	chunks, err := localChunker.ProcessChunk(audioFile, innerChunk)
	if err != nil {
		log.Fatal("Couldn't shrink chunk %#v, got error from localChunker.ProcessChunk : %v", innerChunk, err)
	}
	for _, ch := range chunks {
		chunkLen := ch.End - ch.Start
		if chunkLen > maxLen {
			chs := innerShrink(audioFile, localChunker, ch, absMin, maxLen)
			if len(chs) > 2 {
				//fmt.Println("[debug] tjaffolittio", innerChunk, lastChunker.MinSilenceLen, "=>", localChunker.MinSilenceLen, chs)
				res = append(res, chs...)
				continue
			}
		} else {
			res = append(res, ch)
		}
	}
	return res
}

func shrink(chunker ffmpeg.Chunker, audioFile string, chunk protocol.Chunk, maxLen int64) []protocol.Chunk {
	log.Info("shrink called")
	absMin := int64(math.Floor(float64(chunker.MinSilenceLen) * .5))
	return innerShrink(audioFile, chunker, chunk, absMin, maxLen)
}

// OPTIONS
var (
	maxInnerLen, maxOuterLen, minInnerLen, minOuterLen, missingChunkLen *int64
	innerSplitGap                                                       *int64
	outerSplitGap                                                       = int64(0)
	outerSilenceLen, innerSilenceLen                                    *int64
	project                                                             *string
	sourceDir, annoDir                                                  string
	noWarnings                                                          *bool
	forceSplitPages, forceSplitChunks                                   *bool
	shrinkPages, shrinkChunks                                           *bool
	onePagePerFile                                                      *bool
	insertMissingChunks                                                 bool
)

func main() {

	cmd := "pre_chunker_ffmpeg"

	project = flag.String("project", "", "Project `folder`")
	outerSilenceLen = flag.Int64("outerpause", 1500, "Minimum silence length between pages (milliseconds)")
	innerSilenceLen = flag.Int64("innerpause", 800, "Minimum silence length between chunks within a page (milliseconds)")
	minOuterLen = flag.Int64("minouter", 3000, "Minimum length for pages (milliseconds)")
	maxOuterLen = flag.Int64("maxouter", 120000, "Maximum length for pages (milliseconds)")
	minInnerLen = flag.Int64("mininner", 1000, "Minimum length for chunks within a page (milliseconds)")
	maxInnerLen = flag.Int64("maxinner", 12000, "Maximum length for chunks within a page (milliseconds)")
	//outerSplitGap = flag.Int64("outersplitgap", 0, "Gap between auto-created pages")
	innerSplitGap = flag.Int64("innersplitgap", 100, "Gap between auto-created chunks (milliseconds)")
	forceSplitPages = flag.Bool("forcesplitpages", false, "Force split pages longer than max length")
	forceSplitChunks = flag.Bool("forcesplitchunks", false, "Force split chunks longer than max length")
	shrinkPages = flag.Bool("shrinkpages", false, "Try to shrink pages longer than max length if possible")
	shrinkChunks = flag.Bool("shrinkchunks", false, "Try to shrink pages longer than max length if possible")
	onePagePerFile = flag.Bool("onepageperfile", false, "Pre-paginated input audio files")
	insertMissingChunks = true
	missingChunkLen = flag.Int64("missingchunklen", 3000, "If there is an unchunked part of of at least n milliseconds at the end of a page, create an extra chunk")
	help := flag.Bool("help", false, "Print usage and exit")
	h := flag.Bool("h", false, "Print usage and exit")
	noWarnings = flag.Bool("nowarnings", false, "Suppress warnings")

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

	if _, err := os.Stat(*project); os.IsNotExist(err) {
		log.Fatal("Project folder %s does not exist", *project)
	}

	sourceDir = fmt.Sprintf("%s/source", strings.TrimSuffix(*project, "/"))
	annoDir = fmt.Sprintf("%s/annotation", strings.TrimSuffix(*project, "/"))

	fmt.Fprintf(os.Stderr, "Project: %s\n", *project)
	fmt.Fprintf(os.Stderr, "Source directory: %s\n", sourceDir)
	fmt.Fprintf(os.Stderr, "Annotation directory: %s\n", annoDir)
	fmt.Fprintf(os.Stderr, "Force split pages:  %v\n", *forceSplitPages)
	fmt.Fprintf(os.Stderr, "Force split chunks: %v\n", *forceSplitChunks)
	fmt.Fprintf(os.Stderr, "Shrink pages:  %v\n", *shrinkPages)
	fmt.Fprintf(os.Stderr, "Shrink chunks: %v\n", *shrinkChunks)
	fmt.Fprintf(os.Stderr, "Insert missing chunks: %v\n", insertMissingChunks)
	fmt.Fprintf(os.Stderr, "Outer pause:          %7d\n", *outerSilenceLen)
	fmt.Fprintf(os.Stderr, "Inner pause:          %7d\n", *innerSilenceLen)
	fmt.Fprintf(os.Stderr, "Outer split gap:      %7d\n", outerSplitGap)
	fmt.Fprintf(os.Stderr, "Inner split gap:      %7d\n", *innerSplitGap)
	fmt.Fprintf(os.Stderr, "Min page length:      %7d\n", *minOuterLen)
	fmt.Fprintf(os.Stderr, "Max page length:      %7d\n", *maxOuterLen)
	fmt.Fprintf(os.Stderr, "Min chunk length:     %7d\n", *minInnerLen)
	fmt.Fprintf(os.Stderr, "Max chunk length:     %7d\n", *maxInnerLen)
	fmt.Fprintf(os.Stderr, "Missing chunk length: %7d\n", *missingChunkLen)
	fmt.Fprintf(os.Stderr, "One page per file:    %7v\n", *onePagePerFile)
	fmt.Fprintf(os.Stderr, "Suppress warnings:    %7v\n", *noWarnings)
	fmt.Fprintf(os.Stderr, "\n")

	//pageChunker, err := ffmpeg.NewChunker(*outerSilenceLen), int64(250))
	pageChunker, err := ffmpeg.NewChunker(*outerSilenceLen, int64(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create page chunker: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	innerChunker, err := ffmpeg.NewChunker(*innerSilenceLen, int64(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create inner chunker: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	var aiExtractor ffprobe.InfoExtractor
	if *onePagePerFile {
		aiExtractor, err = ffprobe.NewInfoExtractor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't create info extractor: %v\n", err)
			flag.Usage()
			os.Exit(1)
		}
	}

	nChunks := 0
	nPages := 0
	nFiles := len(flag.Args())

	for _, file := range flag.Args() {
		var audioFile string
		log.Info("Processing %s ", file)
		var pages []protocol.PagePayload
		if strings.HasSuffix(file, ".json") {
			bts, err := os.ReadFile(file)
			if err != nil {
				log.Fatal("couldn't read pages file %s : %v", file, err)
			}
			err = json.Unmarshal(bts, &pages)
			if err != nil {
				log.Fatal("couldn't unmarshal pages file %s : %v", file, err)
			}
		} else if *onePagePerFile {
			audioFile = file
			page, err := audioFile2OnePage(aiExtractor, file)
			if err != nil {
				log.Fatal("Couldn't process audio file %s: %v", file, err)
			}
			pages = []protocol.PagePayload{page}
		} else {
			audioFile = file
			pages, err = audioFile2Pages(file, pageChunker)
			if err != nil {
				log.Fatal("Couldn't process audio file %s: %v", file, err)
			}
		}
		nPages += len(pages)

		for i, page := range pages {
			log.Info("Created page %s %d-%d (%dms)", page.ID, page.Start, page.End, page.End-page.Start)

			// force create folders on first run
			if i == 0 {
				if _, err := os.Stat(annoDir); os.IsNotExist(err) {
					os.Mkdir(annoDir, os.ModePerm)
				}
				if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
					os.Mkdir(sourceDir, os.ModePerm)
				}
			}

			var anno protocol.AnnotationPayload
			if audioFile != "" {
				anno, err = page2Annotation(audioFile, page, innerChunker)
			} else {
				anno, err = page2Annotation(path.Join(sourceDir, page.Audio), page, innerChunker)
			}
			if err != nil {
				log.Fatal("Couldn't process page %s: %v", page.ID, err)
			}
			nChunks += len(anno.Chunks)

			for _, chunk := range anno.Chunks {
				log.Info("Created chunk %s %d-%d (%dms)", chunk.UUID, chunk.Start, chunk.End, chunk.End-chunk.Start)
			}

			json, err := json.MarshalIndent(anno, " ", " ")
			if err != nil {
				log.Fatal("Marshal failed: %v", err)
			}

			annoJSONFile := path.Join(annoDir, fmt.Sprintf("%s.json", page.ID))
			file, err := os.Create(annoJSONFile)
			if err != nil {
				log.Fatal("Couldn't create file %s : %v", annoJSONFile, err)
			}
			defer file.Close()
			file.Write(json)
			log.Info("Saved annotation to file %v", annoJSONFile)
			//fmt.Fprintf(os.Stderr, " => file %s", annoJSONFile)
			//fmt.Fprintf(os.Stderr, ".")
		}

		json, err := json.MarshalIndent(pages, " ", " ")
		if err != nil {
			log.Fatal("Marshal failed: %v", err)
		}

		if audioFile != "" {
			baseName := baseName(audioFile)
			sourceJSONFile := path.Join(sourceDir, fmt.Sprintf("%s.json", baseName))
			file, err := os.Create(sourceJSONFile)
			if err != nil {
				log.Fatal("Couldn't create source file %s : %v", sourceJSONFile, err)
			}
			defer file.Close()
			file.Write(json)
			log.Info("Saved annotation to file %v", sourceJSONFile)
			//fmt.Fprintf(os.Stderr, " %s", sourceJSONFile)
		}
	}
	log.Info("Created %d chunks, %d pages for %d audio file%s", nChunks, nPages, nFiles, appendPluralS(nFiles))
}
