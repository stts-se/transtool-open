package main

import (
	"fmt"
	"os"
	"path"
	//"regexp"
	"encoding/json"
	"flag"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stts-se/transtool/dbapi"
	"github.com/stts-se/transtool/protocol"
	"github.com/stts-se/transtool/validation"
)

// Manually OK:ed files with adjacent identical transcriptions
// TODO Replace with general way of ignoring specific validation
var ignoreEqualTransValidationFor = map[string]bool{
	"STTS-e79649e1-72ea-498b-b16b-63918c48811e09550": true,
	"STTS-ea3c2ba6-f42d-4387-b5f5-cd3acc64b7c702516": true,
	"STTS-94f798b3-6016-4ea5-8d41-4297ae613b3805769": true,
	"STTS-25a8337b-638d-4984-a5fa-1bd6bc3d3c9901358": true,
	"STTS-1760b2b5-6d2a-442e-b9e1-e91c962cfecd09337": true,
}

func ignoreEQTransVal(valRes []validation.ValRes, fn string) []validation.ValRes {
	var res []validation.ValRes

	for _, vr := range valRes {

		if vr.RuleName == "identical_adjacent_transcriptions" && hasIgnorePrefix(fn) {
			fmt.Fprintf(os.Stderr, "Ignoring identical_adjacent_transcriptions:\t%s\n", fn)
		} else {
			res = append(res, vr)
		}
	}

	return res
}

func hasIgnorePrefix(fn string) bool {

	for fn0 := range ignoreEqualTransValidationFor {
		if strings.HasPrefix(fn, fn0) {

			return true
		}
	}

	return false
}

func loadAnnotationData(annotationDataDir string) (map[string]protocol.AnnotationPayload, []dbapi.ValRes, error) {
	res := map[string]protocol.AnnotationPayload{}
	var vRes []dbapi.ValRes
	var errRes error
	files := listJSONFiles(annotationDataDir)
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			bts, err := os.ReadFile(f)
			if err != nil {
				msg := fmt.Sprintf("couldn't read annotation file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, dbapi.ValRes{Level: "error", Message: msg})
				continue
			}
			var annotation protocol.AnnotationPayload
			err = json.Unmarshal(bts, &annotation)
			if err != nil {
				msg := fmt.Sprintf("couldn't unmarshal annotation file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, dbapi.ValRes{Level: "error", Message: msg})
				continue
			}
			// err = validateAnnotation(annotation)
			// if err != nil {
			// 	msg := fmt.Sprintf("invalid json in annotation file %s : %v", f, err)
			// 	//HB errRes = err
			// 	vRes = append(vRes, dbapi.ValRes{Level:"error", Message:msg})
			// 	continue
			// }
			if _, seen := res[annotation.Page.ID]; seen {
				msg := fmt.Sprintf("duplicate page ids for annotation data: %s : %v", f, err)
				errRes = err
				vRes = append(vRes, dbapi.ValRes{Level: "error", Message: msg})
				continue
			}

			//TODO Temp backward compatibility fix NL 20210802
			if annotation.CurrentStatus.Name == "" || annotation.CurrentStatus.Name == "in progress" {
				annotation.CurrentStatus.Name = "normal"
			}

			res[annotation.Page.ID] = annotation
		}
	}
	return res, vRes, errRes
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

type kv struct {
	k string
	v int
}

func main() {

	annotationOnly := flag.Bool("annotation_json_only", false, "Validate only annotation JSON files, ignoring audio and JSON \"source\" files")
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "USAGE: <JSON config file> <sub proj dirs> ...\n")
		fmt.Fprintf(os.Stderr, "\n-annotation_json_only to ignore audio and JSON \"source\" files\n")
		fmt.Fprintf(os.Stderr, "\n(Sample config file in validation/sample_validation_config.json)\n")
		os.Exit(0)

	}

	bts, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config file '%s' : %v", args[0], err)
		os.Exit(1)
	}

	var config validation.Config

	err = json.Unmarshal(bts, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal config : %v", err)
		os.Exit(1)
	}

	validator, err := validation.NewValidator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create new  validation.Validator instance : %v", err)
		os.Exit(1)
	}

	seenFileNames := map[string][]string{}

	//var issues []dbapi.ValRes
	issues := map[dbapi.ValRes]int{}
	for _, dirName := range args[1:] {
		fmt.Println("DIR ", dirName)
		_, err := os.Stat(dirName)
		if err != nil {

			msg := fmt.Sprintf("Failed for dir '%s' : %v\n", dirName, err)
			//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
			issues[dbapi.ValRes{Level: "error", Message: msg}]++
			//fmt.Fprintf(os.Stderr, msg)
			//os.Exit(1)
		}

		sourcePath := path.Join(dirName, "source")
		_, err = os.Stat(sourcePath)
		if err != nil {
			msg := fmt.Sprintf("Failed for dir '%s' : %v\n", sourcePath, err)
			//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
			issues[dbapi.ValRes{Level: "error", Message: msg}]++
			//fmt.Fprintf(os.Stderr, msg)
			//os.Exit(1)
		}

		annotationPath := path.Join(dirName, "annotation")
		_, err = os.Stat(annotationPath)
		if err != nil {
			msg := fmt.Sprintf("Failed for dir '%s' : %v\n", annotationPath, err)
			//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
			issues[dbapi.ValRes{Level: "error", Message: msg}]++
			//fmt.Fprintf(os.Stderr, msg)
			//os.Exit(1)
		}

		var annos map[string]protocol.AnnotationPayload
		if !*annotationOnly {

			db := dbapi.NewDBAPI(dirName, nil)
			vRes, err := db.LoadData()
			if err != nil {
				msg := fmt.Sprintf("Failed loading dir '%s' : %v\n", dirName, err)
				//fmt.Fprintf(os.Stderr, msg)
				//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
				issues[dbapi.ValRes{Level: "error", Message: msg}]++

				//os.Exit(1)
			}
			//issues = append(issues, vRes...)
			for _, vr := range vRes {
				issues[vr]++
			}

			annos = db.GetAnnotationData()
		} else {
			annotationPath := path.Join(dirName, "annotation")
			_, err = os.Stat(annotationPath)
			if err != nil {
				msg := fmt.Sprintf("Failed for dir '%s' : %v\n", annotationPath, err)
				//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
				issues[dbapi.ValRes{Level: "error", Message: msg}]++
				//fmt.Fprintf(os.Stderr, msg)
				//os.Exit(1)
			}
			annos0, vRes, err := loadAnnotationData(annotationPath)
			if err != nil {
				msg := fmt.Sprintf("Failed to load annotation files from '%s': %v", annotationPath, err)
				//issues = append(issues, dbapi.ValRes{Level:"error", Message:msg})
				issues[dbapi.ValRes{Level: "error", Message: msg}]++
				//fmt.Fprintf(os.Stderr, msg)
				//os.Exit(1)

			}
			annos = annos0
			//issues = append(issues, vRes...)
			for _, vr := range vRes {
				issues[vr]++
			}

		}

		pageStatusStats := map[string]int{}
		statusStats := map[string]int{}
		chunkSourceStats := map[string]int{}
		for _, a := range annos {
			fn := a.Page.Audio
			//HB if dirs, ok := seenFileNames[fn]; ok {
				//HB msg := fmt.Sprintf("File name '%s' in multiple dirs %s", fn, strings.Join(dirs, ", ")+", "+dirName)
				//HB issues[dbapi.ValRes{Level: "error", Message: "duplicate_file_name" + "\t" + msg}]++
			//HB}
			seenFileNames[fn] = append(seenFileNames[fn], dirName)

			//if a.CurrentStatus.Name != "" {
			pageStatusStats[a.CurrentStatus.Name]++
			//}

			if a.CurrentStatus.Name == "delete" || a.CurrentStatus.Name == "skip" { //|| a.CurrentStatus.Name == "in progress" {
				continue
			}

			for _, c := range a.Chunks {
				statusStats[c.CurrentStatus.Name]++
				chunkSourceStats[c.CurrentStatus.Source]++

			}
		}

		fmt.Printf("\nPAGE STATUS COUNT\n==============\n")
		for k, v := range pageStatusStats {
			fmt.Printf("'%s'\t%d\n", k, v)
		}

		fmt.Printf("\nCHUNK STATUS COUNT (excluding skip or delete pages)\n==============\n")
		for k, v := range statusStats {
			fmt.Printf("%s\t%d\n", k, v)
		}

		fmt.Printf("\nCHUNK SOURCE COUNT\n==============\n")
		for k, v := range chunkSourceStats {
			fmt.Printf("%s\t%d\n", k, v)
		}

		fmt.Println()

		for fn, anno := range annos {

			if anno.CurrentStatus.Name == "delete" || anno.CurrentStatus.Name == "skip" { //|| a.CurrentStatus.Name == "in progress" {
				continue
			}

			valres := validator.ValidateAnnotation(anno)

			valres = append(valres, validator.IdenticalTranscriptions(anno)...)

			valres = ignoreEQTransVal(valres, fn)

			if len(valres) > 0 {
				fmt.Printf("%s has %d issues\n", fn, len(valres))
				for _, vr := range valres {

					fmt.Printf("%s\t%s\t%s\t%s\n", fn, vr.Level, vr.RuleName, vr.Message) //, vr.ChunkIndex)
					issues[dbapi.ValRes{Level: "validation", Message: vr.Level + "\t" + vr.RuleName /*+ "\t" + vr.Message*/}]++
				}
				fmt.Println()
			}
		}
		//res := validate(config, db)
		//_ = res
	}

	issType := map[string]int{}
	var kvs []kv
	for iss, v := range issues {
		k := fmt.Sprintf("%s\t%s\t", iss.Level, iss.Message)
		kvs = append(kvs, kv{k: k, v: v})
		issType[iss.Level]++
	}

	sort.Slice(kvs, func(i, j int) bool { return kvs[i].v > kvs[j].v })
	if len(kvs) > 0 {
		fmt.Println("=== ISSUES BY FREQUENCY===")
	}
	for _, kv := range kvs {
		fmt.Printf("%s\t%d\n", kv.k, kv.v)
	}
	// if len(kvs) > 0 {
	// 	fmt.Println()
	// }

	// if len(issType) > 0 {
	// 	fmt.Println("=== NUMBER OF ISSUES BY TYPE ===")
	// }
	// for issT, freq := range issType {
	// 	fmt.Printf("%s\t%d\n", issT, freq)
	// }

}

/*
func validate(config validation.Config, db *dbapi.DBAPI) map[string][]validation.ValRes {
	res := map[string][]validation.ValRes{}

	// TODO Better way to list all Pages
	//matchAll := dbapi.Query{TransRE: regexp.MustCompile(".*")}
	//allFiles := db.Search(matchAll)

	annotations := db.GetAnnotationData()

	for _, a := range annotations {
		fmt.Printf("%#v\n", a)
	}

	return res
}
*/
