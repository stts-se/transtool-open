package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	//"strconv"
	"sort"
	"strings"

	"github.com/stts-se/transtool/protocol"
)

// var (
// 	validationChunkMinLen int64 = 1000
// 	validationChunkMaxLen int64 = 6000
// )

type RegexpValidation struct {
	RuleName string `json:"rule_name"`
	Regexp   string `json:"regexp"`
	Level    string `json:"level"`
	Message  string `json:"message"`
}

type Config struct {
	PageStatusNames  string `json:"page_status_names"`
	StatusNames      string `json:"status_names"`
	ValidCharsRegexp string `json:"valid_chars_regexp"`

	LabelPrefix string `json:"label_prefix"`
	LabelSuffix string `json:"label_suffix"`
	Labels      string `json:"labels"`

	TokenSplitRegexp string `json:"token_split_regexp"`

	TransMustMatch    []RegexpValidation `json:"trans_must_match"`
	TransMustNotMatch []RegexpValidation `json:"trans_must_not_match"`
}

var ConfigExample = Config{
	// If empty string is acceptable status name:
	// PageStatusNames:  "normal delete skip, ",
	PageStatusNames:  "normal delete skip",
	StatusNames:      "unchecked ok ok2 skip",
	ValidCharsRegexp: `[\p{L} _.,!?:\]\[-]`,

	LabelPrefix: "[",
	LabelSuffix: "]",

	Labels: "[OVERLAPPING_SPEECH] [UNRECOGNISABLE] [SPEAKER_A] [SPEAKER_B] [BACKGROUND_NOISE]",

	TokenSplitRegexp: `[ \n,.!?]`,
}

var ConfigExample2 = Config{
	// If empty string is acceptable status name:
	// PageStatusNames:  "normal delete skip, ",
	PageStatusNames:  "normal delete skip",
	StatusNames:      "unchecked ok ok2 skip",
	ValidCharsRegexp: `[\p{L} _.,#!?:-]`,

	LabelPrefix: "#",
	LabelSuffix: "",

	//Labels: "#OVERLAPPING_SPEECH #UNRECOGNISABLE #AGENT #CUSTOMER #BACKGROUND_NOISE",
	Labels: "#AGENT #CUSTOMER #OVERLAP #UNKNOWN #NOISE",

	TokenSplitRegexp: `[ \n,.!?]`,

	TransMustMatch: []RegexpValidation{
		{
			RuleName: "trans_initial_label",
			Regexp:   `^\s*#(AGENT|CUSTOMER|OVERLAP|UNKNOWN|NOISE)`,
			Level:    "fatal",
			Message:  "Cannot OK a chunk that doesn't start with one of #AGENT, #CUSTOMER, #OVERLAP, #UNKNOWN or #NOISE",
		},
	},

	TransMustNotMatch: []RegexpValidation{
		{
			RuleName: "repeated_full_stops",
			Regexp:   `[.]\s*[.]`,
			Level:    "error",
			Message:  "Transcription must not include repeated full stops",
		},
	},
}

// Example:  NewValidatorFromJSON([]byte(configExmapleJSON)))
// TODO ignores errors for now
var configExmapleJSON, _ = json.MarshalIndent(ConfigExample, "", "\t")

type regexpValidator struct {
	re       *regexp.Regexp
	ruleName string
	level    string
	message  string
}

type Validator struct {
	config           Config
	pageStatusNames  map[string]bool
	statusNames      map[string]bool
	validCharsRegexp *regexp.Regexp

	labelPrefix string
	labelSuffix string

	labels map[string]bool

	tokenSplitRegexp *regexp.Regexp

	transMustMatch    []regexpValidator
	transMustNotMatch []regexpValidator
}

func NewValidator(c Config) (Validator, error) {
	res := Validator{
		config:          c,
		pageStatusNames: map[string]bool{},
		statusNames:     map[string]bool{},
		labels:          map[string]bool{},
		labelPrefix:     c.LabelPrefix,
		labelSuffix:     c.LabelSuffix,
	}

	splt := regexp.MustCompile("[, ]+")

	if strings.TrimSpace(c.PageStatusNames) == "" {
		return res, fmt.Errorf("Config.PageStatusNames must not be empty")
	}
	for _, s := range splt.Split(c.PageStatusNames, -1) {
		res.pageStatusNames[s] = true
	}

	if strings.TrimSpace(c.StatusNames) == "" {
		return res, fmt.Errorf("Config.StatusNames must not be empty")
	}

	for _, s := range splt.Split(c.StatusNames, -1) {
		res.statusNames[s] = true
	}

	if strings.TrimSpace(c.LabelPrefix) == "" {
		return res, fmt.Errorf("Config.LabelPrefix must not be empty")
	}
	// if strings.TrimSpace(c.LabelSuffix) == "" {
	// 	return res, fmt.Errorf("Config.LabelSuffix must not be empty")
	// }

	if strings.TrimSpace(c.Labels) == "" {
		return res, fmt.Errorf("Config.Labels must not be empty")
	}

	for _, l := range splt.Split(c.Labels, -1) {
		if !strings.HasPrefix(l, res.labelPrefix) {
			return res, fmt.Errorf("label '%s' lacks prefix '%s'", l, res.labelPrefix)
		}
		if !strings.HasSuffix(l, res.labelSuffix) {
			return res, fmt.Errorf("label '%s' lacks suffix '%s'", l, res.labelSuffix)
		}

		res.labels[l] = true
	}

	tokSplit, err := regexp.Compile(c.TokenSplitRegexp)
	if err != nil {
		return res, fmt.Errorf("TokenSplitRegexp failed to compile : %v", err)
	}
	res.tokenSplitRegexp = tokSplit

	validChars, err := regexp.Compile(c.ValidCharsRegexp)
	if err != nil {
		return res, fmt.Errorf("ValidCharsRegexp failed to compile : %v", err)
	}
	res.validCharsRegexp = validChars

	for _, v := range c.TransMustMatch {
		if v.Regexp == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustMatch had empty Regexp")
		}

		if v.RuleName == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustMatch had empty RuleName")
		}

		if v.Level == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustMatch had empty Level")
		}
		if v.Message == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustMatch had empty Message")
		}

		tMatch, err := regexp.Compile(v.Regexp)
		if err != nil {
			return res, fmt.Errorf("validation.NewValidator failed to compile TransMustMatch regexp : %v", err)
		}

		rv := regexpValidator{
			re:       tMatch,
			ruleName: v.RuleName,
			level:    v.Level,
			message:  v.Message,
		}

		res.transMustMatch = append(res.transMustMatch, rv)
	}

	for _, v := range c.TransMustNotMatch {
		if v.Regexp == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustNotMatch had empty Regexp")
		}

		if v.RuleName == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustNotMatch had empty RuleName")
		}

		if v.Level == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustNotMatch had empty Level")
		}
		if v.Message == "" {
			return res, fmt.Errorf("validation.NewValidator failed since RegexpValidation in TransMustNotMatch had empty Message")
		}

		tMatch, err := regexp.Compile(v.Regexp)
		if err != nil {
			return res, fmt.Errorf("validation.NewValidator failed to compile TransMustNotMatch regexp : %v", err)
		}

		rv := regexpValidator{
			re:       tMatch,
			ruleName: v.RuleName,
			level:    v.Level,
			message:  v.Message,
		}

		res.transMustNotMatch = append(res.transMustNotMatch, rv)
	}

	return res, nil
}

func NewValidatorFromJSON(configJSON []byte) (Validator, error) {
	var c Config
	err := json.Unmarshal(configJSON, &c)
	if err != nil {
		return Validator{}, fmt.Errorf("NewValidatorFromJSON failed to unmashal JSON : %v", err)
	}
	return NewValidator(c)
}

func (v *Validator) ValidateAnnotation(a protocol.AnnotationPayload) []ValRes {
	var res []ValRes

	status := a.CurrentStatus.Name
	if !v.pageStatusNames[status] {
		vr := ValRes{
			RuleName:   "unknown_page_status",
			Level:      "error",
			Message:    fmt.Sprintf("Unknown page status: '%s'", status),
			ChunkIndex: -1,
		}
		res = append(res, vr)
	}

	res = append(res, validateAnnotationPayload(v.statusNames, a)...)
	for _, c := range a.Chunks {
		//res = append(res, ValidateTransChunk(c)...)
		// TODO only validate OK/OK2?
		if strings.HasPrefix(c.CurrentStatus.Name, "ok") {
			res = append(res, v.ValidateTrans(c.Trans)...)
		}
	}

	return res
}

func (v *Validator) ValidateTrans(t string) []ValRes {
	var res []ValRes
	res = append(res, validateTransChars(v.validCharsRegexp, v.labels, t)...)
	res = append(res, validateInTransLabels(v.labelPrefix, v.labelSuffix, v.tokenSplitRegexp, v.labels, t)...)

	for _, rv := range v.transMustMatch {
		if !rv.re.MatchString(t) {
			vr := ValRes{
				RuleName:   rv.ruleName,
				Level:      rv.level,
				Message:    rv.message,
				ChunkIndex: -1,
			}
			res = append(res, vr)
		}
	}

	for _, rv := range v.transMustNotMatch {
		if rv.re.MatchString(t) {
			vr := ValRes{
				RuleName:   rv.ruleName,
				Level:      rv.level,
				Message:    rv.message,
				ChunkIndex: -1,
			}
			res = append(res, vr)
		}
	}

	res = remDupes(res)

	return res
}

func remDupes(valRes []ValRes) []ValRes {
	var res []ValRes
	seen := map[string]bool{}

	for _, vr := range valRes {
		s := vr.Level + vr.RuleName + vr.Message + fmt.Sprintf("%d", vr.ChunkIndex)
		if !seen[s] {
			res = append(res, vr)
		}
		seen[s] = true
	}

	return res
}

func (v *Validator) IdenticalTranscriptions(a protocol.AnnotationPayload) []ValRes {
	var res []ValRes

	seenTrans := map[string]map[int]bool{}

	for i, c := range a.Chunks {
		t := c.Trans
		t = strings.TrimSpace(t)

		if ignoreTrans(t, v) {
			continue
		}

		if _, ok := seenTrans[t]; !ok {
			m := map[int]bool{}
			seenTrans[t] = m
		}
		seenTrans[t][i] = true
	}

	for k, v := range seenTrans {
		if len(v) > 1 {
			indx := []int{}
			for i := range v {
				indx = append(indx, i+1)
			}

			sort.Slice(indx, func(i, j int) bool { return indx[i] < indx[j] })

			adjacent := false
			for i, n := range indx {
				// consecutive chunks
				if i > 0 && n-indx[i-1] == 1 {
					adjacent = true
					break
				}
			}

			msg := fmt.Sprintf("%s\tidentical transcriptions: '%s'\tchunks %v", a.Page.Audio, k, indx)

			if adjacent {
				vr := ValRes{Level: "warning", RuleName: "identical_adjacent_transcriptions", Message: msg}
				res = append(res, vr)
				//} else {
				//	vr := ValRes{Level: "info", RuleName: "identical_transcriptions", Message: msg}
				//	res = append(res, vr)

			}
		}
	}

	return res
}

func ignoreTrans(trans string, v *Validator) bool {

	// Skip empty transcriptions, since these should be catched by other rules
	trans = strings.TrimSpace(trans)
	if trans == "" {
		return true
	}

	toks := v.tokenSplitRegexp.Split(trans, -1)

	nLabels := 0
	for _, t := range toks {
		if strings.HasPrefix(t, v.labelPrefix) {
			nLabels++
		}
	}

	if len(toks)-nLabels < 4 {
		return true
	}

	// Only labels, no proper transcription
	if nLabels == len(toks) {
		return true
	}

	return false
}

func (v *Validator) Config() Config { return v.config }

type Validation struct {
	Result []ValRes `json:"result"`
}

type ValRes struct {
	RuleName   string `json:"rule_name"`
	Level      string `json:"level"`
	ChunkIndex int    `json:"chunk_index"`
	Message    string `json:"message"`
}

func validateAnnotationPayload(validStatusNames map[string]bool, a protocol.AnnotationPayload) []ValRes {
	var res []ValRes

	res0 := validateChunks(validStatusNames, a)
	res = append(res, res0...)

	//TODO Validate missing status
	// Validate missing transcription (if not labelled empty)
	// Validate length of transcription compared to length of chunk
	// Spelling?
	// Validate markup

	// TODO You might want to validate concurrently if many tests are done

	return res
}

func validateChunks(validStatusNames map[string]bool, a protocol.AnnotationPayload) []ValRes {
	var res []ValRes
	// Look for overlapping chunks
	for i, c := range a.Chunks {
		if i == 0 {
			continue
		}

		// Look for overlapping chunks
		prev := a.Chunks[i-1]
		if prev.End > c.Start {

			msg := fmt.Sprintf("Overlapping chunks: the end time of previous chunk is higher than the start time of the current one: %d vs %d", prev.End, c.Start)
			v := ValRes{Level: "fatal",
				RuleName:   "overlapping_chunks",
				ChunkIndex: i,
				Message:    msg,
			}
			res = append(res, v)
		}
	}

	// Validate chunk in isolation
	for i, c := range a.Chunks {
		res = append(res, validateChunk(i, validStatusNames, c)...)
	}

	return res
}

//var statusName = map[string]bool{
//	"ok":        true,
//	"ok2":       true,
//	"skip":      true,
//	"unchecked": true,
//}

func validateChunk(i int, validStatusNames map[string]bool, c protocol.TransChunk) []ValRes {
	var res []ValRes

	if strings.TrimSpace(c.Trans) == "" && strings.HasPrefix(c.CurrentStatus.Name, "ok") {

		msg := "Cannot OK a chunk without transcription. Transcribe or skip."
		if i > -1 {
			msg = fmt.Sprintf("Cannot OK a chunk without transcription. Transcribe or skip chunk no. %d", i+1)
		}
		vr := ValRes{
			RuleName:   "ok_without_transcription",
			ChunkIndex: i,
			Level:      "error",
			Message:    msg,
		}
		res = append(res, vr)
	}

	if c.Start >= c.End {
		msg := "Chunk has start time that is greater or equal to its ends time."
		if i > -1 {
			msg = fmt.Sprintf("Chunk no. %d has start time that is greater or equal to its ends time.", i+1)
		}
		vr := ValRes{
			RuleName:   "start_greater_than_end_time",
			ChunkIndex: i,
			Level:      "error",
			Message:    msg,
		}
		res = append(res, vr)
	}

	// dur := c.End - c.Start
	// if dur < validationChunkMinLen {
	// 	vr := ValRes{
	// 		ChunkIndex: i,
	// 		Level:      "warning",
	// 		Message:    fmt.Sprintf("chunk is shorter than %v ms", validationChunkMinLen),
	// 	}
	// 	res = append(res, vr)
	// }
	// if dur > validationChunkMaxLen {
	// 	vr := ValRes{
	// 		ChunkIndex: i,
	// 		Level:      "warning",
	// 		Message:    fmt.Sprintf("chunk is longer than %v ms", validationChunkMaxLen),
	// 	}
	// 	res = append(res, vr)
	// }

	if !validStatusNames[c.CurrentStatus.Name] {

		msg := fmt.Sprintf("Chunk has unknown status name '%s'", c.CurrentStatus.Name)
		if i > -1 {
			msg = fmt.Sprintf("Chunk no. %d has unknown status name '%s'", i+1, c.CurrentStatus.Name)
		}

		vr := ValRes{
			RuleName:   "unknown_chunk_status",
			ChunkIndex: i,
			Level:      "error",
			Message:    msg,
		}
		res = append(res, vr)

	}

	return res
}

// TODO Validating a single TransChunk returns ChunkIndex = -1
func (v *Validator) ValidateTransChunk(c protocol.TransChunk) []ValRes {
	return validateChunk(-1, v.statusNames, c)
}

// TODO return index for illegal chars, so that it could be highlighted?
func validateTransChars(allowedChars *regexp.Regexp, labels map[string]bool, trans string) []ValRes {
	var res []ValRes

	// chars in labels are by definition valid
	for l := range labels {
		trans = strings.ReplaceAll(trans, l, "")
	}

	// remove all valid chars, so that only invalid ones are left

	validCharsRemoved := allowedChars.ReplaceAllString(trans, "")

	for _, invalid := range validCharsRemoved {

		msg := fmt.Sprintf("Invalid char in transcription: '%s'", string(invalid))
		fmt.Println("validateTransChars: ", msg)
		vr := ValRes{
			RuleName:   "invalid_chars",
			Level:      "error",
			Message:    msg,
			ChunkIndex: -1,
		}

		res = append(res, vr)
	}

	return res
}

func validateInTransLabels(labelPrefix, labelSuffix string, tokenSplitPattern *regexp.Regexp, validLabels map[string]bool, trans string) []ValRes {

	var res []ValRes

	toks := tokenSplitPattern.Split(trans, -1)
	for _, t := range toks {

		if strings.HasPrefix(t, labelPrefix) || (labelSuffix != "" && strings.HasSuffix(t, labelSuffix)) {
			if !validLabels[t] {

				var vls []string
				for k, _ := range validLabels {
					vls = append(vls, k)
				}
				msg := fmt.Sprintf("Invalid label: '%s'. Valid labels: %s", t, strings.Join(vls, ", "))
				vr := ValRes{
					RuleName:   "invalid_label",
					Level:      "error",
					ChunkIndex: -1,
					Message:    msg,
				}

				res = append(res, vr)
			}
		}
	}

	return res
}
