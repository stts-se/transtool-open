package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	//"github.com/stts-se/transtool/dbapi"
	"github.com/stts-se/transtool/protocol"
)

type kv struct {
	k string
	v int
}

func listJSONFiles(dir string) []string {
	var res []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	for _, f := range files {
		if !f.IsDir() {
			if filepath.Ext(f.Name()) == ".json" {
				res = append(res, path.Join(dir, f.Name()))
			}
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

func LoadAnnotationData(path string) (map[string]protocol.AnnotationPayload, error) {
	res := map[string]protocol.AnnotationPayload{}
	files := listJSONFiles(path)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "WARNING: No JSON files in %s\n", path)
	}

	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			bts, err := os.ReadFile(f)
			if err != nil {
				return res, fmt.Errorf("couldn't read annotation file %s : %v", f, err)
			}
			var annotation protocol.AnnotationPayload
			err = json.Unmarshal(bts, &annotation)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshal annotation file %s : %v", f, err)
			}
			if _, seen := res[annotation.Page.ID]; seen {
				return res, fmt.Errorf("duplicate page ids for annotation data: %s", annotation.Page.ID)
			}

			//TODO Temp backward compatibility fix NL 20210802
			if annotation.CurrentStatus.Name == "" || annotation.CurrentStatus.Name == "in progress" {
				annotation.CurrentStatus.Name = "normal"
			}

			res[annotation.Page.ID] = annotation
		}
	}
	return res, nil
}

type Stats struct {
	PagePerEditor map[string]int
	PageStatus    map[string]int
}

func status(a protocol.AnnotationPayload) string {
	if a.CurrentStatus.Name == "skip" || a.CurrentStatus.Name == "delete" {
		return a.CurrentStatus.Name
	}

	cStatus := map[string]int{}
	for _, c := range a.Chunks {
		s := c.CurrentStatus.Name
		s = strings.TrimSpace(s)
		cStatus[s]++
	}

	// All chunks have same status
	if len(cStatus) == 1 {
		for cs := range cStatus {
			return cs
		}
	}

	if cStatus["unchecked"] > 0 {
		return "unchecked"
	}

	if cStatus["skip"] > 0 {
		return "skip"
	}

	return "UNKNOWN_STATUS"
}

func editors(a protocol.AnnotationPayload) []string {
	eds := map[string]bool{}
	for _, c := range a.Chunks {

		if c.CurrentStatus.Name != "unchecked" {
			s := c.CurrentStatus.Source
			s = strings.TrimSpace(s)
			eds[s] = true
		}
	}

	var res []string
	for e := range eds {
		res = append(res, e)
	}

	return res
}

func okeyedChunks(a protocol.AnnotationPayload) (int, int) {
	var okey int
	var tot int
	for _, c := range a.Chunks {
		tot++
		if strings.HasPrefix(c.CurrentStatus.Name, "ok") {
			okey++
		}
	}
	return okey, tot
}

func okejedMillis(a protocol.AnnotationPayload) int64 {
	var res int64

	for _, c := range a.Chunks {
		if strings.HasPrefix(c.CurrentStatus.Name, "ok") {
			t := c.End - c.Start

			res += t
		}
	}

	return res
}

func timestamp(a protocol.AnnotationPayload) string {
	if a.CurrentStatus.Name == "delete" || a.CurrentStatus.Name == "skip" {
		return a.CurrentStatus.Timestamp
	}

	//nChunks := len(a.Chunks)
	return a.Chunks[0].CurrentStatus.Timestamp
}

func main() {

	t := time.Now().In(time.Local)
	fmt.Println(t)

	pageStatus := map[string]int{}
	pageEditors := map[string]map[string]int{}
	date := map[string]int{}
	//deletePerEditor := map[string]int{}

	var totTimeOKMillis int64

	for _, dirName := range os.Args[1:] {
		b := filepath.Base(dirName)
		fmt.Printf("%s\t", b)
		_, err := os.Stat(dirName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed for dir '%s' : %v\n", dirName, err)
			os.Exit(1)
		}

		annotationPath := path.Join(dirName, "annotation")
		_, err = os.Stat(annotationPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed for dir '%s' : %v\n", annotationPath, err)
			os.Exit(1)
		}

		annotationFiles, err := LoadAnnotationData(annotationPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load annotation dir '%s': %v\n", annotationPath, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Loaded %d JSON files\n", len(annotationFiles))

		for _, a := range annotationFiles {
			s := status(a)

			totTimeOKMillis += okejedMillis(a)

			// print skipped files with many OK:ed chunks
			if s == "skip" {
				okey, tot := okeyedChunks(a)

				if okey > 0 && tot-okey < 4 {
					fmt.Printf("OK/tot chunks in SKIP:\t%s\t%d\t%d\n", a.Page.Audio, okey, tot)
				}
			}

			if s != "unchecked" {
				ts := strings.Split(timestamp(a), " ")
				d := ts[0]
				date[d]++
				//fmt.Println(date)
			}

			//if s != "" {
			//fmt.Println(s)
			pageStatus[s]++
			//}

			if a.CurrentStatus.Name == "delete" || a.CurrentStatus.Name == "skip" {
				e := a.CurrentStatus.Source
				if _, ok := pageEditors[e]; !ok {
					pageEditors[e] = map[string]int{}
				}
				pageEditors[e][a.CurrentStatus.Name]++
				continue
			}

			for _, e := range editors(a) {
				//fmt.Printf("%v\n", e)
				if _, ok := pageEditors[e]; !ok {
					pageEditors[e] = map[string]int{}
				}
				pageEditors[e][s]++
			}

		}

	}

	pStatus := []kv{}

	for s, n := range pageStatus {
		pStatus = append(pStatus, kv{k: s, v: n})
	}

	sort.Slice(pStatus, func(i, j int) bool { return pStatus[i].v > pStatus[j].v })

	pEditors := []kv{}
	for e, s := range pageEditors {
		n := 0
		for _, n0 := range s {
			n = n + n0
		}
		pEditors = append(pEditors, kv{k: e, v: n})
	}

	sort.Slice(pEditors, func(i, j int) bool { return pEditors[i].v > pEditors[j].v })

	fmt.Println("=== STATUS/PAGE ===")
	for _, s := range pStatus {
		fmt.Printf("%s\t%d\n", s.k, s.v)
	}

	fmt.Println()

	fmt.Printf("%18s\tTOT\tOK\tDELETE\tSKIP\n", "=== EDITOR ===")
	for _, e := range pEditors {
		fmt.Printf("%18s\t%d\t%d\t%d\t%d\n", e.k, e.v, pageEditors[e.k]["ok"], pageEditors[e.k]["delete"], pageEditors[e.k]["skip"])
	}

	fmt.Printf("\n\nTime in minutes of OK:ed chunks: %.2f\n\n", (float64(totTimeOKMillis)/1000.0)/60.0)

	dates := []kv{}
	for k, v := range date {
		dates = append(dates, kv{k: k, v: v})
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].k > dates[j].k })

	nDates := 14
	if len(dates) <= nDates {
		fmt.Println("DATES")
	} else {
		fmt.Printf("LAST %d DATES\n", nDates)
	}
	for i, kv := range dates {
		if i >= nDates {
			break
		}
		fmt.Printf("%s\t%d\n", kv.k, kv.v)
	}
	if len(dates) >= nDates {
		fmt.Println("...")
	}
}
