package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

//func dmmy() { fmt.Println() }

// fmt.Println(NewValidatorFromJSON([]byte(configExmapleJSON)))

var validStatusNames = map[string]bool{
	"ok":        true,
	"skip":      true,
	"unchecked": true,
}

func TestAnnotationFile01(t *testing.T) {

	tf1bts, err := os.ReadFile("annotation_test_file_01.json")
	if err != nil {
		t.Errorf("READ FAIL! : %v", err)
	}

	var anno protocol.AnnotationPayload
	err = json.Unmarshal(tf1bts, &anno)
	if err != nil {
		t.Errorf("JSON UNMARSHAL FAIL : %v", err)
	}

	valRes := validateAnnotationPayload(validStatusNames, anno)
	if len(valRes) == 0 {
		t.Errorf("Expected validation results, got empty list")
	}

	// Overlapping chunks: "start": 320 after "end": 36230 in chunk before
	vr0 := valRes[0]
	rn0 := "overlapping_chunks"
	if w, g := rn0, vr0.RuleName; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}

	if w, g := "fatal", vr0.Level; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}
	if w, g := 2, vr0.ChunkIndex; w != g {
		t.Errorf("expected '%d' got '%d'", w, g)
	}

	// status "ok" but not transcription
	vr1 := valRes[1]
	rn1 := "ok_without_transcription"
	if w, g := rn1, vr1.RuleName; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}

	if w, g := "error", vr1.Level; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}
	if w, g := 0, vr1.ChunkIndex; w != g {
		t.Errorf("expected '%d' got '%d'", w, g)
	}

	// unknown status name
	vr2 := valRes[2]
	rn2 := "unknown_chunk_status"
	if w, g := rn2, vr2.RuleName; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}

	if w, g := "error", vr2.Level; w != g {
		t.Errorf("expected '%s' got '%s'", w, g)
	}
	if w, g := 1, vr2.ChunkIndex; w != g {
		t.Errorf("expected '%d' got '%d'", w, g)
	}

	//fmt.Printf("%#v\n", valRes)
}

func TestAnnotationFile02(t *testing.T) {

	tf2bts, err := os.ReadFile("annotation_test_file_02.json")
	if err != nil {
		t.Errorf("READ FAIL! : %v", err)
	}

	var anno protocol.AnnotationPayload
	err = json.Unmarshal(tf2bts, &anno)
	if err != nil {
		t.Errorf("JSON UNMARSHAL FAIL : %v", err)
	}

	valRes := validateAnnotationPayload(validStatusNames, anno)
	if len(valRes) == 0 {
		t.Errorf("Expected validation results, got empty list")
	}

	if w, g := 1, len(valRes); w != g {
		t.Errorf("Wanted %d got %d", w, g)
	}

	vr := valRes[0]
	if w, g := "start_greater_than_end_time", vr.RuleName; w != g {
		t.Errorf("Wanted '%s' got '%s'", w, g)
	}

}

func TestValidateTransChars(t *testing.T) {
	validChars := regexp.MustCompile(`[\p{Latin} .,!?:-]`)
	valres := validateTransChars(validChars, map[string]bool{}, `Verboten: #%&"¿`)
	if w, g := 5, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	valres = validateTransChars(validChars, map[string]bool{}, `All OK æøł`)
	if w, g := 0, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}
}

func TestValidateTransChars2(t *testing.T) {
	validChars := regexp.MustCompile(`[\p{Latin} #.,!?:-]`)
	valres := validateTransChars(validChars, map[string]bool{}, `Verboten: %&"¿`)
	if w, g := 4, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	valres = validateTransChars(validChars, map[string]bool{}, `All OK æøł #`)
	if w, g := 0, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}
}

func TestValidateInTransLabels(t *testing.T) {
	tokenSplitPattern := regexp.MustCompile(`[ \n,.!?]`)
	prefix := "["
	suffix := "]"

	validLabels := map[string]bool{"[UNRECOGNISABLE]": true}

	trans := "interesting stuff [UNRECOGNISABLE] is ok, but [UNRECOGNISABLE ] and [UNRECOGNIZZZABLE] are not"

	valres := validateInTransLabels(prefix, suffix, tokenSplitPattern, validLabels, trans)
	if w, g := 3, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}
}

func TestValidator1(t *testing.T) {
	// See validator.go var configExmaple
	validator, err := NewValidatorFromJSON(configExmapleJSON)
	if err != nil {
		t.Errorf("failed to create Validator instance : %v", err)
	}

	vres1 := validator.ValidateTrans("fnöske")
	if w, g := 0, len(vres1); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	vres2 := validator.ValidateTrans("fnöske [OVERLAPPING_SPEECH broken label")

	if w, g := 1, len(vres2); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	vres3 := validator.ValidateTrans("fnöske [UNKNOWN_LABEL_ZZZ] undefined label")
	//fmt.Printf("%#v\n", vres3)
	if w, g := 1, len(vres3); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	tf1bts, err := os.ReadFile("annotation_test_file_01.json")
	if err != nil {
		t.Errorf("READ FAIL! : %v", err)
	}

	var anno protocol.AnnotationPayload
	err = json.Unmarshal(tf1bts, &anno)
	if err != nil {
		t.Errorf("JSON UNMARSHAL FAIL : %v", err)
	}
	vres4 := validator.ValidateAnnotation(anno)
	if w, g := 4, len(vres4); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	vres4_0 := vres4[0]
	if w, g := "unknown_page_status", vres4_0.RuleName; w != g {
		t.Errorf("wanted %s got %s", w, g)
	}

	//fmt.Println(vres4_0.Message)

}

func TestValidator2(t *testing.T) {
	v, err := NewValidator(ConfigExample2)
	if err != nil {
		t.Errorf("failed to create new validator : %v", err)
	}

	valres := v.ValidateTrans("#AGENT är tillåtet")
	if w, g := 0, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	valres = v.ValidateTrans("#AGENT_X är inte tillåtet")
	if w, g := 1, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	valres = v.ValidateTrans("MÅSTE starta med label")
	if w, g := 1, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	valres = v.ValidateTrans("MÅSTE starta med label och får inte .. ha två punkter i rad")
	if w, g := 2, len(valres); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	//fmt.Printf("%#v\n", valres)
}

var cfgLabels = Config{
	PageStatusNames:  "delete skip",
	StatusNames:      "ok skip",
	ValidCharsRegexp: `[\p{L} _.\[\],#!?:-]`,
	LabelPrefix:      "[",
	LabelSuffix:      "]",
	Labels:           "[q:backround-noise] [o:overlapping-spech] [u:unrecognisable]",
	TokenSplitRegexp: `[ \n,.!?]`,
}

func TestValidator3(t *testing.T) {
	v, err := NewValidator(cfgLabels)
	if err != nil {
		t.Errorf("%v", err)
	}

	vRes1 := v.ValidateTrans("Akka pakka [q:backround-noise] sss")

	for _, vr := range vRes1 {
		fmt.Printf("%v\n", vr)
	}

	if len(vRes1) != 0 {
		t.Errorf("Expected 0 got %d", len(vRes1))
	}
}
