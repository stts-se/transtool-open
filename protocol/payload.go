package protocol

import (
	"encoding/json"
)

type PagePayload struct {
	Chunk
	ID    string `json:"id"`
	Audio string `json:"audio"`
}

type SplitRequestPayload struct {
	Audio string `json:"audio"`
	// LeftContext in milliseconds
	LeftContext int64 `json:"left_context"`
	// RightContext in milliseconds
	RightContext int64 `json:"right_context"`
	Chunk        Chunk `json:"chunk"`
}

type Chunk struct {
	// Start time in milliseconds
	Start int64 `json:"start"`
	// End time in milliseconds
	End int64 `json:"end"`
}

type TransChunk struct {
	UUID string `json:"uuid"`
	Chunk
	Trans         string   `json:"trans"`          //`json:"trans,omitempty"`
	CurrentStatus Status   `json:"current_status"` //`json:"current_status,omitempty"`
	StatusHistory []Status `json:"status_history"` //`json:"status_history,omitempty"`
}

type AnnotationWithAudioData struct {
	AnnotationPayload
	Chunk
	// Base64Audio is a base64 string representation of the audio
	Base64Audio string `json:"base64audio,omitempty"`
	FileType    string `json:"file_type"`
	Offset      int64  `json:"offset"`
}

func (aa AnnotationWithAudioData) PrettyMarshal() ([]byte, error) {
	copy := aa
	copy.Base64Audio = ""
	return json.Marshal(copy)
}

// Annotation

type Status struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

type AnnotationPayload struct {
	SubProj       string       `json:"sub_proj"`
	Page          PagePayload  `json:"page,omitempty"`
	Chunks        []TransChunk `json:"chunks"`
	Labels        []string     `json:"labels,omitempty"`
	CurrentStatus Status       `json:"current_status,omitempty"`
	StatusHistory []Status     `json:"status_history,omitempty"`
	Comment       string       `json:"comment,omitempty"`
	Index         int64        `json:"index,omitempty"`
}

// func (tc *TransChunk) SetCurrentStatus(s Status) {
// 	if tc.CurrentStatus.Name != "" || tc.CurrentStatus.Source != "" {
// 		tc.StatusHistory = append(tc.StatusHistory, tc.CurrentStatus)
// 	}
// 	tc.CurrentStatus = s
// }

// func (ap *AnnotationPayload) SetCurrentStatus(s Status) {
// 	if ap.CurrentStatus.Name != "" || ap.CurrentStatus.Source != "" {
// 		ap.StatusHistory = append(ap.StatusHistory, ap.CurrentStatus)
// 	}
// 	ap.CurrentStatus = s
// }

type UnlockPayload struct {
	SubProj string `json:"sub_proj"`
	PageID  string `json:"page_id"`
	//ClientID string `json:"client_id"`
	//UserName string `json:"user_name"`
}

// QueryPayload holds criteria used to search in the database
type QueryPayload struct {
	//ClientID     string       `json:"client_id"`
	//UserName     string       `json:"user_name"`
	Request      QueryRequest `json:"request"`
	StepSize     int64        `json:"step_size"`
	RequestIndex string       `json:"request_index"`
	CurrID       string       `json:"curr_id"`
	Context      int64        `json:"context,omitempty"`
}

type ValIssue struct {
	HasIssue  bool     `json:"has_issue"`
	RuleNames []string `json:"rule_names"`
}

type QueryRequest struct {
	PageStatus      string   `json:"page_status,omitempty"`
	Status          string   `json:"status,omitempty"`
	AudioFile       string   `json:"audio_file,omitempty"`
	Source          string   `json:"source,omitempty"`
	TransRE         string   `json:"trans_re,omitempty"`
	ValidationIssue ValIssue `json:"validation_issue,omitempty"`
	//	transRECompiled *regexp.Regexp
}

type MatchingPage struct {
	MatchingChunks []int             `json:"matching_chunks"`
	Page           AnnotationPayload `json:"page"`
}

type QueryResult struct {
	MatchingPages []MatchingPage `json:"matching_pages"`
}

type ListFiles struct {
	SubProj string `json:"sub_proj"`
}

// ASR
type ASROutputChunk struct {
	Chunk
	Text string `json:"text"`
}

type ASROutput struct {
	Chunks []ASROutputChunk
}

type ASRConfig struct {
	URL          string `json:"url"`
	Lang         string `json:"lang"`
	Encoding     string `json:"encoding"`
	SampleRate   int    `json:"sample_rate"`
	ChannelCount int    `json:"channels"`
}

type ASRRequest struct {
	SubProj string `json:"sub_proj"`
	PageID  string `json:"page_id"`
	Lang    string `json:"lang"`
	Chunk   Chunk  `json:"chunk"`
	UUID    string `json:"uuid"`
}

type ASRResponse struct {
	//PageID string `json:"page_id"`
	UUID string `json:"uuid"`
	Text string `json:"text"`
}
