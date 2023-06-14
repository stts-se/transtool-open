package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	//	"github.com/rsc/getopt"

	"github.com/stts-se/transtool/abbrevs"
	"github.com/stts-se/transtool/dbapi"
	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/modules"
	"github.com/stts-se/transtool/modules/ffmpeg"
	"github.com/stts-se/transtool/modules/ffprobe"
	"github.com/stts-se/transtool/protocol"
	"github.com/stts-se/transtool/validation"
)

// Message for sending to client
type Message struct {
	MessageType string `json:"message_type"`
	Payload     string `json:"payload"`

	// Fatal is a non-recoverable error (GUI will be disabled)
	Fatal string `json:"fatal,omitempty"`

	// Error is a recoverable error (GUI will be cleared, but not disabled)
	Error string `json:"error,omitempty"`

	// Info is for informational messages, nothing is disabled or cleared
	Info string `json:"info,omitempty"`
}

type AnnotationUnlockAndQueryPayload struct {
	Annotation  protocol.AnnotationPayload `json:"annotation"`
	Unlock      protocol.UnlockPayload     `json:"unlock"`
	Query       protocol.QueryPayload      `json:"query"`
	ReturnAudio bool                       `json:"return_audio"`
}

func getParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

var chunkExtractor ffmpeg.ChunkExtractor

// print serverMsg to server log, and return an http error with clientMsg and the specified error code (http.StatusInternalServerError, etc)
func httpError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
	log.Error(serverMsg)
	http.Error(w, clientMsg, errCode)
}

// print serverMsg to server log, send client message over websocket
func wsError(conn *websocket.Conn, serverMsg string, clientMsg string) {
	log.Error(serverMsg)
	payload := Message{
		Error: clientMsg,
	}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
		log.Error(msg)
		return
	}
	clientMutex.Lock()
	defer clientMutex.Unlock()

	err = conn.WriteMessage(websocket.TextMessage, resJSON)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

// print serverMsg to server log, send client message as non-recoverable error message over websocket
func wsFatal(conn *websocket.Conn, serverMsg string, clientMsg string) {
	log.Error(serverMsg)
	payload := Message{
		Fatal: clientMsg,
	}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
		log.Error(msg)
		return
	}
	clientMutex.Lock()
	defer clientMutex.Unlock()

	err = conn.WriteMessage(websocket.TextMessage, resJSON)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

// print serverMsg to server log, send error message as json to client
func jsonError(w http.ResponseWriter, serverMsg string, clientMsg string) {
	log.Error(serverMsg)
	payload := Message{
		Error: clientMsg,
	}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(resJSON))
}

var clientMutex sync.RWMutex
var clients = make(map[dbapi.ClientID]*websocket.Conn)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func keepAlive() {
	msg := Message{
		MessageType: "keep_alive",
	}
	jsn, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("failed to marshal JSON : %v", err)
		return
	}

	t := time.NewTicker(23 * time.Second)
	for range t.C {

		clientMutex.Lock()
		for id, ws := range clients {
			err = ws.WriteMessage(websocket.TextMessage, jsn)
			if err != nil {
				log.Info("[main] websocket error for client ID '%s' : %v", id, err)
				ws.Close()
				delete(clients, id)
				proj.UnlockAll(id)
			}
		}
		clientMutex.Unlock()
	}

	// clean up pages locked by not active IDs
	unlockOrphanedLockedPages()
	go pushStats() // To update Locked numbers
}

// clean up pages locked by not active IDs
func unlockOrphanedLockedPages() {
	lockedBy := proj.UserIDsForLockedPages()
	clientMutex.RLock()
	defer clientMutex.RUnlock()
	for _, id := range lockedBy {
		_, ok := clients[id]
		if !ok {
			proj.UnlockAll(id)
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//HB test (second is pw)
	//name, _, loggedin := r.BasicAuth()
	//fmt.Printf("Testing.. username:%s loggedin:%t\n", name, loggedin)
	//HB end

	clientID := vars["client_id"]
	if clientID == "" {
		msg := "Expected client ID, got empty string"
		jsonError(w, msg, msg)
		return
	}
	userName := vars["user_name"]

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to upgrade HTTP request to websocket : %v", err)
		jsonError(w, msg, msg)
		return
	}

	clientMutex.Lock()
	defer clientMutex.Unlock()
	for clID, oldws := range clients {
		if clID.UserName == userName && clID.ID == clientID {

			msg := Message{
				MessageType: "keep_alive",
			}
			jsn, err := json.Marshal(msg)
			if err != nil {
				log.Fatal("failed to marshal JSON : %v", err)
				return
			}

			//clientMutex.Lock()
			err = oldws.WriteMessage(websocket.TextMessage, jsn)
			//clientMutex.Unlock()
			if err != nil {
				log.Info("[main] websocket error for OLD client ID '%s' : %v", clID, err)
				oldws.Close()
				//clientMutex.Lock()
				delete(clients, clID)
				proj.UnlockAll(clID)
				//clientMutex.Unlock()
			} else {
				msg := fmt.Sprintf("User %s is already logged in\nMaybe you are logged in in another location?", userName)
				wsFatal(ws, msg, msg)
				ws.Close() // the client will close the websocket if needed (to avoid double error messages from server)
				return
			}
		}
	}

	clID := dbapi.ClientID{ID: clientID, UserName: userName}
	//clientMutex.Lock()
	clients[clID] = ws
	//clientMutex.Unlock()
	log.Info("[main] Added websocket for client id %s", clID)

	// listen forever
	go listenToClient(ws, clID)
}

func wsPayload(conn *websocket.Conn, msgType string, payload interface{}) {
	bts, err := json.Marshal(payload)
	if err != nil {
		log.Error("failed to marshal struct into JSON : %v", err)
		return
	}
	resp := Message{
		//ClientID:    msg.ClientID,
		MessageType: msgType,
		Payload:     string(bts),
	}

	jsnMsg, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}
	clientMutex.Lock()
	defer clientMutex.Unlock()
	err = conn.WriteMessage(websocket.TextMessage, jsnMsg)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

func wsPayloadAllClients(msgType string, payload interface{}) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for _, conn := range clients {

		bts, err := json.Marshal(payload)
		if err != nil {
			log.Error("failed to marshal struct into JSON : %v", err)
			return
		}
		resp := Message{
			//ClientID:    msg.ClientID,
			MessageType: msgType,
			Payload:     string(bts),
		}

		jsnMsg, err := json.Marshal(resp)
		if err != nil {
			msg := fmt.Sprintf("Failed to marshal struct into JSON : %v", err)
			wsError(conn, msg, msg)
			return
		}
		err = conn.WriteMessage(websocket.TextMessage, jsnMsg)
		if err != nil {
			log.Error("Couldn't write to conn: %v", err)
		}
	}
}

func wsInfo(conn *websocket.Conn, msg string) {
	resp := Message{
		//ClientID:    msg.ClientID,
		Info: msg,
	}

	jsnMsg, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}

	clientMutex.Lock()
	defer clientMutex.Unlock()
	err = conn.WriteMessage(websocket.TextMessage, jsnMsg)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

// When a websocket connection is estabilshed, the first thing that
// happens is that the server sends a sub-project listing and a
// validation.Config to the client
func listenToClient(conn *websocket.Conn, clientID dbapi.ClientID) {
	//wsInfo(conn, "Websocket created on server")

	if strings.TrimSpace(clientID.ID) == "" {
		msg := fmt.Sprintf("empty ID field: %#v", clientID)
		wsError(conn, msg, msg)
		return
	}

	if strings.TrimSpace(clientID.UserName) == "" {
		msg := fmt.Sprintf("empty UserName field: %#v", clientID)
		wsError(conn, msg, msg)
		return
	}

	//TODO better names, not full paths?
	//dirListing := proj.ListSubProjsWithProgress()
	dirListing := proj.ListSubProjs()
	dirNames := strings.Join(dirListing, ":")
	//res := db.ProjectName()

	//HB added 4/10 2021
	if *cfg.EnableAutoplay {
		wsPayload(conn, "enable_autoplay", *cfg.EnableAutoplay)
	}
	if *cfg.NoDelete {
		wsPayload(conn, "no_delete", *cfg.NoDelete)
	}
	//end HB

	wsPayload(conn, "project_name", dirNames)

	valCfg := validator.Config()

	wsPayload(conn, "validation_config", valCfg)

	stats := proj.Stats()
	wsPayload(conn, "stats", stats)

	editorNames := proj.GetStatusSources()
	wsPayload(conn, "editor_names", editorNames)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			emptyMsg := Message{}
			if msg != emptyMsg {

				msg := fmt.Sprintf("listenToClient: Websocket error for Message '%#v' : %v", msg, err)
				log.Error(msg)
			}
			clientMutex.Lock()
			delete(clients, clientID)
			proj.UnlockAll(clientID)
			clientMutex.Unlock()
			go pushStats()
			log.Info("[main] Removed websocket for client id %s", clientID)

			return
		}

		//log.Info("[main] Payload received over websocket: %#v\n", msg)

		switch msg.MessageType {
		case "stats":
			stats := proj.Stats()
			wsPayload(conn, "stats", stats)

		case "saveunlockandnext":
			var payload AnnotationUnlockAndQueryPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("saveunlockandnext: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			saveUnlockAndNext(conn, clientID, payload)
			go pushStats()

		case "save":
			var payload protocol.AnnotationPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("save: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			save(conn, payload)
			go pushStats()

		case "unlock":
			var payload protocol.UnlockPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("unlock: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}

			err = proj.Unlock(payload.SubProj, payload.PageID, clientID)
			if err != nil {
				msg := fmt.Sprintf("Couldn't unlock page: %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			msg := fmt.Sprintf("Unlocked page %s for user %v", payload.PageID, clientID)
			wsPayload(conn, "explicit_unlock_completed", msg)
			go pushStats()

		case "unlock_all":
			var payload protocol.UnlockPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("unlock_all: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}

			n, err := proj.UnlockAll(clientID)
			if err != nil {
				msg := fmt.Sprintf("Failed to unlock : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			msg := fmt.Sprintf("Unlocked %d page%s for user %s", n, pluralS(n), clientID.UserName)
			wsPayload(conn, "explicit_unlock_completed", msg)
			go pushStats()

		case "asr-request":
			log.Info("[main] asr-request called!\n")

			//HB
			//if !googleASR.Initialised() {
			//	msg := "ASR not initialised on server."
			//	log.Error(msg)
			//	wsError(conn, msg, msg)
			//	return
			//}
			//END HB

			var payload protocol.ASRRequest
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("asr-request: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			log.Info("[main] payload: %#v", payload)
			//HB
			gCloudASR(conn, payload)

		case "validate":
			var payload protocol.AnnotationPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("validate: Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}
			validate(conn, payload)

		case "validate_trans":
			var payload string //protocol.TransChunk
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("validate_trans_chunk: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			validateTrans(conn, payload)

		case "list-db-audio-files-request":
			var payload protocol.ListFiles
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("list-db-audio-files-request: Failed to unmarshal payload : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			if payload.SubProj == "" {
				msg := "no value for 'sub_proj' in 'list-db-audio-files-request' payload"
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			res, err := proj.ListAudioFiles(payload.SubProj)
			if err != nil {
				msg := fmt.Sprintf("massage type list-db-audio-files-request error : %v", err)
				log.Error(msg)
				wsError(conn, msg, msg)
				return
			}
			wsPayload(conn, "list-db-audio-files-response", res)

			// NL 20210716 commenting out
			// "audio_from_page", since it doesn't appear
			// to be used.  After adding the dbapi.Proj
			// layer aroud dbap.DBAPI, you now need to
			// supply the SubProj value, which needs adding below

		// case "audio_from_page":

		// 	//var payload protocol.TransChunk
		// 	var payload protocol.PagePayload

		// 	err := json.Unmarshal([]byte(msg.Payload), &payload)
		// 	if err != nil {
		// 		msg := fmt.Sprintf("audio_from_page: Failed to unmarshal payload : %v", err)
		// 		wsError(conn, msg, msg)
		// 		return
		// 	}
		// 	//fmt.Printf("INCOMING 1:\t%#v\n", msg.Payload)
		// 	//fmt.Printf("INCOMING 2:\t%#v\n", payload)
		// 	//p, err := json.Marshal(payload)
		// 	//fmt.Printf("INCOMING 3:\t%s\n", p)
		// 	res, err := audioFromPage(protocol.AnnotationPayload{Page: payload}, -1)
		// 	wsPayload(conn, "audio_for_page_response", res)

		default:
			log.Error("Unknown message type: %s", msg.MessageType)
		}
	}
}

// TODO initialise validator on cmd line
func validate(conn *websocket.Conn, payload protocol.AnnotationPayload) {
	vres := validator.ValidateAnnotation(payload)
	if len(vres) > 0 {
		//fmt.Printf("VALIDATION: %#v\n", vres)
		wsPayload(conn, "validation_result", validation.Validation{Result: vres})
	}
}

func validateTrans(conn *websocket.Conn, payload string) {
	valRes := validator.ValidateTrans(payload)
	if len(valRes) > 0 {
		wsPayload(conn, "trans_validation_result", validation.Validation{Result: valRes})
	}
}

// TODO Global instaniated in main: var gcloudAsr modules.GoogleASR
func gCloudASR(conn *websocket.Conn, payload protocol.ASRRequest) {

	page, err := proj.PageFromID(payload.SubProj, payload.PageID)
	if err != nil {
		msg := fmt.Sprintf("db.PageFromID error: %v", err)
		log.Error(msg)
		message := Message{
			Error: msg,
		}
		m, err := json.Marshal(message)
		if err != nil {
			log.Error("gCloudASR failed to marshal %#v", message)
			return
		}
		clientMutex.Lock()
		defer clientMutex.Unlock()
		err = conn.WriteMessage(websocket.TextMessage, m)
		if err != nil {
			log.Error("gCloudASR couldn't write message to conn: %v", err)
		}
		return
	}
	audioPath, err := proj.BuildAudioPath(payload.SubProj, page.Audio)
	if err != nil {
		msg := fmt.Sprintf("gCloudASR: db.BuildAudioPath error: %v", err)
		log.Error(msg)
		message := Message{
			Error: msg,
		}
		m, err := json.Marshal(message)
		if err != nil {
			log.Error("gCloudASR failed to marshal %#v", message)
			return
		}
		clientMutex.Lock()
		defer clientMutex.Unlock()
		err = conn.WriteMessage(websocket.TextMessage, m)
		if err != nil {
			log.Error("gCloudASR couldn't write message to conn: %v", err)
		}
		return
	}

	info, err := aiExtractor.Process(audioPath)
	if err != nil {
		msg := fmt.Sprintf("aiExtractor.Process error: %v", err)
		log.Error(msg)
		message := Message{
			Error: msg,
		}
		m, err := json.Marshal(message)
		if err != nil {
			log.Error("gCloudASR failed to marshal %#v", message)
			return
		}
		clientMutex.Lock()
		defer clientMutex.Unlock()
		err = conn.WriteMessage(websocket.TextMessage, m)
		if err != nil {
			log.Error("gCloudASR couldn't write message to conn: %v", err)
		}
		return
	}
	config := protocol.ASRConfig{
		URL:          *cfg.ASRURL,
		Lang:         payload.Lang,
		Encoding:     strings.TrimPrefix(filepath.Ext(page.Audio), "."),
		SampleRate:   int(info.SampleRate),   // 48000,
		ChannelCount: int(info.ChannelCount), //2,
	}

	chnk := protocol.Chunk{Start: payload.Chunk.Start, End: payload.Chunk.End}

	log.Info("[main] audio: %s", audioPath)
	log.Info("[main] chunk: %#v", chnk)
	log.Info("[main] asr config: %#v", config)

	//HB
	//res, err := googleASR.Process(config, audioPath, chnk)
	//res, err := sttsASR.Process(config, audioPath, chnk)

	//var err []string
	var res protocol.ASROutput
	
	if payload.Lang == "sv-SE" {
		res, err = sttsASR.Process(config, audioPath, chnk)
	} else if payload.Lang == "ga-IE" {
		res, err = abairASR.Process(config, audioPath, chnk)
	} else {
		if googleASR.Initialised() {
			res, err = googleASR.Process(config, audioPath, chnk)
		} else {
			err = fmt.Errorf("googleASR not initialised")
		}
	}
	//END HB

	if err != nil {
		//HB
		//msg := fmt.Sprintf("googleASR.Process error: %v", err)
		msg := fmt.Sprintf("ASR Process error: %v", err)
		//END HB
		log.Error(msg)
		message := Message{
			Error: msg,
		}
		m, err := json.Marshal(message)
		if err != nil {
			log.Error("gCloudASR failed to marshal %#v", message)
			return
		}
		clientMutex.Lock()
		defer clientMutex.Unlock()

		err = conn.WriteMessage(websocket.TextMessage, m)
		if err != nil {
			log.Error("gCloudASR couldn't write message to conn: %v", err)
		}
		return
	}

	var recog []string
	for _, chnk := range res.Chunks {
		recog = append(recog, chnk.Text)
	}

	txt := strings.ToLower(strings.Join(recog, " "))

	resp := protocol.ASRResponse{UUID: payload.UUID, Text: txt}

	msg := fmt.Sprintf("Got ASR result: %s", txt)
	log.Info("[main] %v", msg)

	wsPayload(conn, "asr-response", resp)

}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pushStats() {
	wsPayloadAllClients("stats", proj.Stats())
	clientMutex.RLock()
	n := len(clients)
	clientMutex.RUnlock()

	log.Info("[main] Pushed stats to all clients (%d)", n)
}

const fallbackContext = int64(0)

func load(conn *websocket.Conn, annotation protocol.AnnotationPayload, explicitContext int64) error {
	var context int64
	if explicitContext > 0 {
		context = explicitContext
		// } else if ctx, ok := contextMap[annotation.PageType]; ok {
		// 	context = ctx
	} else {
		context = fallbackContext
	}

	audioPath, err := proj.BuildAudioPath(annotation.SubProj, annotation.Page.Audio)
	if err != nil {
		return fmt.Errorf("load: couldn't build audio path: %v", err)
	}
	request := protocol.SplitRequestPayload{
		Audio:        audioPath,
		Chunk:        protocol.Chunk{Start: annotation.Page.Start, End: annotation.Page.End},
		LeftContext:  context,
		RightContext: context,
	}
	res, err := chunkExtractor.ProcessFileWithContext(request, "")
	if err != nil {
		serverMsg := fmt.Sprintf("Chunk extractor failed : %v", err)
		wsError(conn, serverMsg, fmt.Sprintf("Chunk extractor failed for %s. See server log for details.", request.Audio))
		return nil
	}
	res.AnnotationPayload = annotation
	res.Start = request.Chunk.Start
	res.End = request.Chunk.End

	wsPayload(conn, "audio_chunk", res)
	return nil
}

// NL 20210615: Code below extracted from func load(...)
func audioFromPage(annotation protocol.AnnotationPayload, explicitContext int64) (protocol.AnnotationWithAudioData, error) {
	res := protocol.AnnotationWithAudioData{}

	//fmt.Printf("ANNOTATION IN:\t%#v\n", annotation)

	var context int64
	if explicitContext > 0 {
		context = explicitContext
		// } else if ctx, ok := contextMap[annotation.PageType]; ok {
		// 	context = ctx
	} else {
		context = fallbackContext
	}

	audioPath, err := proj.BuildAudioPath(annotation.SubProj, annotation.Page.Audio)
	if err != nil {
		return res, fmt.Errorf("audioFromPage: couldn't build audio path: %v", err)
	}
	request := protocol.SplitRequestPayload{
		Audio:        audioPath,
		Chunk:        protocol.Chunk{Start: annotation.Page.Start, End: annotation.Page.End},
		LeftContext:  context,
		RightContext: context,
	}
	res, err = chunkExtractor.ProcessFileWithContext(request, "")
	if err != nil {
		//serverMsg := fmt.Sprintf("Chunk extractor failed : %v", err)
		//wsError(conn, serverMsg, fmt.Sprintf("Chunk extractor failed for %s. See server log for details.", request.Audio))
		return res, fmt.Errorf("chunk extractor failed : %v", err)
	}
	res.AnnotationPayload = annotation
	res.Start = request.Chunk.Start
	res.End = request.Chunk.End

	//wsPayload(conn, "audio_chunk", res)
	return res, nil
}

func save(conn *websocket.Conn, payload protocol.AnnotationPayload) {
	var err error
	if payload.Page.ID == "" {
		msg := fmt.Sprintf("Missing page id for annotation data : %v", payload)
		wsError(conn, msg, msg)
		return
	}

	//log.Info("[main] save | %#v", payload)

	// save annotation
	err = proj.Save(payload)
	if err != nil {
		msg := fmt.Sprintf("Failed to save annotation : %v", err)
		wsError(conn, msg, msg)
		return
	}
	log.Info("[main] Saved annotation for page id %s", payload.Page.ID)
	msg := fmt.Sprintf("Saved annotation for page id %s", payload.Page.ID)
	wsInfo(conn, msg)
	validate(conn, payload)

	//updateSubProjListings()
}

func saveUnlockAndNext(conn *websocket.Conn, clientID dbapi.ClientID, payload AnnotationUnlockAndQueryPayload) {
	var err error
	if payload.Annotation.Page.ID != "" && payload.Annotation.Page.ID != payload.Unlock.PageID {
		msg := fmt.Sprintf("Mismatching ids for annotation/unlock data : %v/%v", payload.Annotation.Page.ID, payload.Unlock.PageID)
		wsError(conn, msg, msg)
		return
	}

	log.Info("[main] saveUnlockAndNext | AnnotationPaylod ID: '%s'", payload.Annotation.Page.ID)

	var savedAnnotation protocol.AnnotationPayload

	// save annotation
	if payload.Annotation.Page.ID != "" {
		err = proj.Save(payload.Annotation)
		if err != nil {
			msg := fmt.Sprintf("Failed to save annotation : %v", err)
			wsError(conn, msg, msg)
			return
		}
		log.Info("[main] Saved annotation %s", payload.Annotation.Page.ID)
		msg := fmt.Sprintf("Saved annotation for page with id %s", payload.Annotation.Page.ID)
		wsInfo(conn, msg)
		savedAnnotation = payload.Annotation

	}

	//go updateSubProjListings()

	// get next
	query := payload.Query
	if strings.TrimSpace(clientID.UserName) == "" {
		msg := "User name not provided for query"
		wsError(conn, msg, msg)
		return
	}
	if strings.TrimSpace(clientID.ID) == "" {
		msg := "Client ID not provided for query"
		wsError(conn, msg, msg)
		return
	}

	if query.StepSize == 0 && query.RequestIndex == "" {
		msg := "Neither step size nor request index was provided for query"
		wsError(conn, msg, msg)
		return
	}
	if query.CurrID == "undefined" {
		query.CurrID = ""
	}

	// if payload.Unlock.SubProj == "" {
	// 	msg := fmt.Sprintf("Missing value in payload for unlock.sub_proj")
	// 	wsError(conn, msg, msg)
	// 	return
	// }

	if payload.Annotation.SubProj == "" {
		msg := "Missing value in payload for annotation.sub_proj"
		wsError(conn, msg, msg)
		return
	}

	aPage, msg, err := proj.GetNextPage(payload.Annotation.SubProj, query, payload.Unlock.PageID, clientID, true)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		wsError(conn, msg, msg)
		return
	}
	if msg == "" || aPage.Page.ID != "" {
		// NL 20210609: i load fylls ljudet i, *och* skrivs till klienten
		//err = load(conn, aPage, query.Context)
		//if err != nil {
		//	msg := fmt.Sprintf("Couldn't load page : %v", err)
		//	wsError(conn, msg, msg)
		//}

		//fmt.Printf("PAGE AUDIO:\t%#v\n", aPage.Page.Audio)

		// NL 20210615: TMP: same as before, but split into separate steps

		if payload.ReturnAudio {
			annoWithAudio, err := audioFromPage(aPage, query.Context)
			if err != nil {
				msg := fmt.Sprintf("saveUnlockAndNext: failed to extract audio for page : %v", err)
				log.Info(msg)
				wsError(conn, msg, msg)
				return

			}

			wsPayload(conn, "audio_chunk", annoWithAudio)
		} else {
			//NL 20210615: Use this to return page without audio:
			wsPayload(conn, "annotation_no_audio", aPage)
		}
	} else {
		msgFmted := ""
		if msg != "" {
			msgFmted = fmt.Sprintf(": %s", msg)
		}
		if query.RequestIndex != "" {
			reqI, err := strconv.Atoi(query.RequestIndex)
			if err == nil {
				msg := fmt.Sprintf("Couldn't go to page %d%s", (reqI + 1), msgFmted)
				wsPayload(conn, "no_audio_chunk", msg)
			} else {
				msg := fmt.Sprintf("Couldn't go to %s page%s", query.RequestIndex, msgFmted)
				wsPayload(conn, "no_audio_chunk", msg)
			}
		} else {
			direction := "next"
			if query.StepSize < 0 {
				direction = "previous"
			}
			//msg := fmt.Sprintf("Couldn't find any %s pages matching status %v%s", direction, query.RequestStatus, msgFmted)
			msg := fmt.Sprintf("Couldn't find any %s pages%s", direction, msgFmted)
			wsPayload(conn, "no_audio_chunk", msg)
		}
		if savedAnnotation.Page.ID != "" {
			err = load(conn, savedAnnotation, query.Context)
			if err != nil {
				msg := fmt.Sprintf("Couldn't load annotation: %v", err)
				wsError(conn, msg, msg)
			}
		}

		return
	}

	// unlock entry
	if payload.Unlock.PageID != "" {
		err = proj.Unlock(payload.Annotation.SubProj, payload.Unlock.PageID, clientID)
		if err != nil {
			msg := fmt.Sprintf("Couldn't unlock page: %v", err)
			wsError(conn, msg, msg)
			return
		}
		msg := fmt.Sprintf("Unlocked page %s for user %s", payload.Unlock.PageID, clientID.UserName)
		wsInfo(conn, msg)
	}

}

var walkedURLs []string

func generateDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<html><head><title>%s</title></head><body>", "STTS Transtool: Doc")
	for _, url := range walkedURLs {
		fmt.Fprintf(w, "%s<br/>\n", url)
	}

	protoc, err := generateProtocolDoc()
	if err != nil {
		log.Error("generateDoc() failed generateProtocolDoc() : %v", err)
	}

	fmt.Fprint(w, protoc)

	fmt.Fprint(w, "</body></html>")
}

type Pair struct {
	Name string
	JSN  string
}

func ptcl() ([]Pair, error) {
	var res []Pair

	p1, err := json.MarshalIndent(AnnotationUnlockAndQueryPayload{}, "", " ")
	if err != nil {
		return res, err
	}
	res = append(res, Pair{"saveunlockandnext", string(p1)})

	return res, nil
}

func generateProtocolDoc() (string, error) {
	//var res string

	tmplStr := `<p><h1>Protocol structs</h1><p>{{ range . }} 
<h2>{{ .Name }}</h2><pre>{{ .JSN }}</pre>
{{ end }}`

	t := template.New("protocol")

	tmpl, err := t.Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("generateProtocolDoc() failed to parse HTML template : %v", err)
	}

	ptcl, err := ptcl()
	if err != nil {
		return "", fmt.Errorf("generateProtocolDoc() failed to marshal example structs : %v", err)
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, ptcl)
	if err != nil {
		return "", fmt.Errorf("generateProtocolDoc() failed to execute template : %v", err)
	}

	return tpl.String(), nil
}

func serveAudio(w http.ResponseWriter, r *http.Request) {
	subProj := getParam("sub_proj", r)
	file := getParam("file", r)

	msg := ""
	if subProj == "" {
		msg = "missing 'sub_proj' parameter to serve_audio call "
	}
	if file == "" {
		msg += "missing 'file' parameter to serve_audio call "
	}

	if msg != "" {
		log.Error("serveAudio: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	//http.ServeFile(w, r, path.Join(*cfg.ProjectDir, "audio", file))
	http.ServeFile(w, r, path.Join(subProj, "audio", file))
}

func hasASR(w http.ResponseWriter, r *http.Request) {
	//HB
	//fmt.Fprintf(w, "%v\n", googleASR.Initialised())
	fmt.Fprintf(w, "%v\n", true)
	//END HB
}

func addProject(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	subProj0 := params["subproj"]
	//fmt.Fprintf(w, "Requesting load of new sub project %v\n", subProj0)

	if *cfg.ProjectRoot == "" {
		msg := "error: Project root is not defined"
		log.Error("addProject: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	subProj := path.Join(*cfg.ProjectRoot, subProj0)
	err := proj.AddProj(subProj, &validator)
	if err != nil {
		msg := fmt.Sprintf("error: Add failed: %v", err)
		log.Error("addProject: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	valRes, err := proj.LoadData(subProj)
	if err != nil {
		msg := fmt.Sprintf("error: Load failed: %v", err)
		log.Error("addProject: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	levelFreq := map[string]int{}
	for _, vr := range valRes {
		log.Warning("%s\t%s", vr.Level, vr.Message)
		levelFreq[vr.Level]++
	}
	if len(levelFreq) > 0 {
		log.Warning("\n\n===== NUMBER OF ISSUES ======\n")
	}
	for k, v := range levelFreq {
		log.Warning("%s\t%d", k, v)
	}

	fmt.Fprintf(w, "Loaded new sub project %v\n", subProj0)

	dirNames := strings.Join(proj.ListSubProjs(), ":")
	wsPayloadAllClients("project_name", dirNames)
	wsPayloadAllClients("stats", proj.Stats())
}

func reloadProject(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	subProj0 := params["subproj"]
	log.Info("[main] Requesting reload of sub project %v", subProj0)
	//fmt.Fprintf(w, "Requesting reload of sub project %v\n", subProj0)
	subProj := path.Join(*cfg.ProjectRoot, subProj0)
	_, err := proj.LoadData(subProj)
	if err != nil {
		msg := fmt.Sprintf("error: Reload failed: %v", err)
		log.Error("reloadProject: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Reloaded sub project %v\n", subProj0)

	dirNames := strings.Join(proj.ListSubProjs(), ":")
	wsPayloadAllClients("project_name", dirNames)
	wsPayloadAllClients("stats", proj.Stats())
}

func unloadProject(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	subProj0 := params["subproj"]
	log.Info("[main] Requesting unload of sub project %v", subProj0)
	//fmt.Fprintf(w, "Requesting unload of sub project %v\n", subProj0)
	subProj := path.Join(*cfg.ProjectRoot, subProj0)
	err := proj.UnloadData(subProj)
	if err != nil {
		msg := fmt.Sprintf("error: Unload failed: %v", err)
		log.Error("unloadProject: " + msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Unloaded sub project %v\n", subProj0)

	dirNames := strings.Join(proj.ListSubProjs(), ":")
	wsPayloadAllClients("project_name", dirNames)
	wsPayloadAllClients("stats", proj.Stats())
}

type PayloadSlice struct {
	Value []string `json:"value"`
}

func listProjects(w http.ResponseWriter, r *http.Request) {
	log.Info("[main] Requesting project listing")
	projNames := []string{}
	for _, p := range proj.ListSubProjs() {
		projNames = append(projNames, filepath.Base(p))
	}
	payload := PayloadSlice{Value: projNames}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "%s", string(resJSON))
}

func reloadValidationConfig(w http.ResponseWriter, r *http.Request) {

	remote := r.RemoteAddr
	if !(strings.Contains(remote, "localhost") || strings.Contains(remote, "127.0.0.")) {
		msg := fmt.Sprintf("/reload_config_file only allowed from localhost/127.0.0.- called from '%s'", remote)
		http.Error(w, msg+"\n", http.StatusBadRequest)
		return
	}

	vConfBts, err := os.ReadFile(*cfg.ValidationConfigFile)
	if err != nil {
		msg := fmt.Sprintf("Failed to read validation config file : %v", err)
		log.Error("%s", msg)

		http.Error(w, msg+"\n", http.StatusInternalServerError)
		return
	}
	var conf validation.Config
	err = json.Unmarshal(vConfBts, &conf)
	if err != nil {
		msg := fmt.Sprintf("Failed to unmarshal validation config file '%s' : %v", *cfg.ValidationConfigFile, err)
		log.Error(msg)
		http.Error(w, msg+"\n", http.StatusInternalServerError)
		return
	}

	v, err := validation.NewValidator(conf)
	if err != nil {

		msg := fmt.Sprintf("Failed to create validation.Validator : %v", err)
		log.Error(msg)
		http.Error(w, msg+"\n", http.StatusInternalServerError)
		return
	}
	validator = v

	fmt.Fprintf(w, "Loaded validation config file:\n%s\n", string(vConfBts))

	//newValidator, err := validation.NewValidator(valCfgFileName)
}

var cfg = &Config{}
var proj *dbapi.Proj //db *dbapi.DBAPI

//var clientIDProjMutex = &sync.Mutex{}
//var clientIDProj = map[string]dbapi.DBAPI{}

// Config for server
type Config struct {
	Protocol             *string `json:"protocol"`
	Host                 *string `json:"host"`
	Port                 *string `json:"port"`
	StaticDir            *string `json:"static_dir"`
	BlockAudio           *bool   `json:"block_audio"`
	ProjectDirs          *string `json:"project_dir"`
	ProjectRoot          *string `json:"project_root"`
	Debug                *bool   `json:"debug"`
	Ffmpeg               *string `json:"ffmpeg"`
	GCloudCredentials    *string `json:"gcloud_credentials"`
	AbbrevDir            *string `json:"abbrev_dir"`
	ValidationConfigFile *string `json:"validation_config_file"`

	//HB added 4/10 2021
	NoDelete       *bool `json:"no_delete"`
	EnableAutoplay *bool `json:"enable_autoplay"`

	// HL added 20230530
	ASRURL *string `json:"asr_url"`
}

// HB
var googleASR modules.GoogleASR
var abairASR modules.AbairASR
var sttsASR modules.SttsASR

// END HB
var aiExtractor ffprobe.InfoExtractor
var validator validation.Validator

func main() {

	cmd := path.Base(os.Args[0])

	// Flags
	cfg.Host = flag.String("host", "localhost", "Server `host`")
	cfg.Port = flag.String("port", "7372", "Server `port`")
	cfg.StaticDir = flag.String("static", "static", "Serve static `directory`")
	cfg.BlockAudio = flag.Bool("block_audio", false, "Block audio dir from being served")
	//cfg.ProjectDir = flag.String("project", "", "Project `directory`")
	cfg.ProjectDirs = flag.String("project_dirs", "", "Project directories separated by ':' (path1/dir1:path1/dir2 [...])")
	cfg.ProjectRoot = flag.String("project_root", "", "Project root for dynamic project creation")
	cfg.Ffmpeg = flag.String("ffmpeg", "ffmpeg", "Ffmpeg command/`path`")
	cfg.GCloudCredentials = flag.String("gcloud_credentials", "", "Google Cloud ASR credentials file path")
	//NL 20210715 cfg.AbbrevDir = flag.String("abbrev_dir", "{projectdir}/../abbreviation_files", "Abbreviation files `directory`")
	cfg.AbbrevDir = flag.String("abbrev_dir", "", "Abbreviation files `directory`")
	cfg.Debug = flag.Bool("debug", false, "Debug mode")
	protocol := "http"
	cfg.Protocol = &protocol

	//HB added 4/10 2021
	cfg.NoDelete = flag.Bool("no_delete", false, "Disallow deletion of chunks (useful for some types of project)")
	cfg.EnableAutoplay = flag.Bool("enable_autoplay", false, "Enable autoplay (experimental)")

	cfg.ValidationConfigFile = flag.String("validation_config", "", "Validation config JSON file path. Example file: validation/sample_validation_config.json")

	cfg.ASRURL = flag.String("asr_url", "http://localhost:8887/recognise", "ASR `URL`")

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

	if strings.HasPrefix(*cfg.ProjectDirs, "-") {
		fmt.Fprintf(os.Stderr, "Invalid project dirs: %s\n", *cfg.ProjectDirs)
		flag.Usage()
		os.Exit(1)
	}

	if strings.HasPrefix(*cfg.GCloudCredentials, "-") {
		fmt.Fprintf(os.Stderr, "Invalid google credentials file: %s\n", *cfg.GCloudCredentials)
		flag.Usage()
		os.Exit(1)
	}

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Didn't expect cmd line args except for flags, found: %#v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	if *cfg.ProjectDirs == "" {
		fmt.Fprintf(os.Stderr, "Required flag project_dirs not set\n")
		flag.Usage()
		os.Exit(1)
	}

	if *cfg.GCloudCredentials == "" {
		fmt.Fprintf(os.Stderr, "\nASR NOT ACTIVATED. Required flag gcloud_credentials not set\n\n")
		//flag.Usage()
		//os.Exit(1)
	}

	_, err := os.Stat(*cfg.StaticDir)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "\nERROR: Required flag static points to a non-existing dir: %s\n", *cfg.StaticDir)
		flag.Usage()
		os.Exit(1)
	}

	if *cfg.ValidationConfigFile == "" {
		fmt.Fprintf(os.Stderr, "\nWARNING: NO VALIDATION CONFIG FILE. Flag validation_config not set. Using defaults.\n\n")
		//flag.Usage()
		//os.Exit(1)
		v, err := validation.NewValidator(validation.ConfigExample2)
		if err != nil {
			log.Fatal("Failed to create validation.Validator : %v", err)
		}
		// Declared above as a global
		validator = v
	} else {

		vConfBts, err := os.ReadFile(*cfg.ValidationConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read validation config file : %v\n", err)
			os.Exit(1)
		}
		var conf validation.Config
		err = json.Unmarshal(vConfBts, &conf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal validation config file '%s' : %v\n", *cfg.ValidationConfigFile, err)
			os.Exit(1)
		}

		v, err := validation.NewValidator(conf)
		if err != nil {
			log.Fatal("Failed to create validation.Validator : %v", err)
		}
		// Declared above as a global
		validator = v

	}
	cfgJSON, _ := json.MarshalIndent(cfg, "", "\t")
	log.Info("[main] Server config:\n%s\n\n", string(cfgJSON))

	vDatorcfgJSON, _ := json.MarshalIndent(validator.Config(), "", "\t")
	log.Info("[main] Validator config:\n%s\n\n", string(vDatorcfgJSON))

	proj0, err := dbapi.NewProj(*cfg.ProjectDirs, &validator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load project dir : %v", err)
		os.Exit(1)
	}
	proj = &proj0

	//NL 20210715 abbrevDir := strings.Replace(*cfg.AbbrevDir, "{projectdir}", *cfg.ProjectDir, -1)
	//os.MkdirAll(abbrevDir, os.ModePerm)
	//NL 20210715 abbrevManager = abbrevs.NewAbbrevManager(abbrevDir)
	abbrevManager = abbrevs.NewAbbrevManager(*cfg.AbbrevDir)
	err = abbrevManager.Load()
	if err != nil {
		log.Fatal("failed to load abbrevs : %v\nUse -abbrev_dir to set abbreviations directory", err)
		os.Exit(1)
	}

	if len(abbrevManager.Lists()) == 0 {
		log.Warning("!!! No abbreviation lists for '%s'", *cfg.ProjectDirs)
	}

	ffmpeg.FfmpegCmd = *cfg.Ffmpeg
	chunkExtractor, err = ffmpeg.NewChunkExtractor()
	if err != nil {
		log.Fatal("Couldn't initialize chunk extractor: %v", err)
	}

	if *cfg.GCloudCredentials != "" {
		googleASR, err = modules.NewGoogleASR(*cfg.GCloudCredentials)
		if err != nil {
			log.Warning("Failed to initialise GCloud ASR: %v", err)
		}
	}

	abairASR, err = modules.NewAbairASR()
	if err != nil {
		log.Warning("Failed to initialise Abair ASR: %v", err)
	}
	sttsASR, err = modules.NewSttsASR()
	if err != nil {
		log.Warning("Failed to initialise Stts ASR: %v", err)
	}

	aiExtractor, err = ffprobe.NewInfoExtractor()
	if err != nil {
		log.Fatal("Failed to initialise AI extractor: %v", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/doc/", generateDoc).Methods("GET")
	r.HandleFunc("/ws/{client_id}/{user_name}", wsHandler)
	if !*cfg.BlockAudio {
		r.HandleFunc("/audio/{file}", serveAudio).Methods("GET")
	}

	r.HandleFunc("/admin/unload/{subproj}", unloadProject)
	r.HandleFunc("/admin/reload/{subproj}", reloadProject)
	r.HandleFunc("/admin/load/{subproj}", addProject)
	r.HandleFunc("/admin/list_projects", listProjects)

	docs := make(map[string]string)
	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		t, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		if info, ok := docs[t]; ok {
			t = fmt.Sprintf("%s - %s", t, info)
		}
		walkedURLs = append(walkedURLs, t)
		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("Failure to walk URLs : %v", err)
		log.Fatal(msg)
	}

	info, err := os.Stat(*cfg.StaticDir)
	if os.IsNotExist(err) {
		log.Fatal("Static dir %s does not exist", *cfg.StaticDir)
	}
	if !info.IsDir() {
		log.Fatal("Static dir %s is not a directory", *cfg.StaticDir)
	}

	r.HandleFunc("/has_asr", hasASR)

	r.HandleFunc("/abbrev/list_lists", listLists)
	r.HandleFunc("/abbrev/list_lists_with_length", listListsWithLength)
	r.HandleFunc("/abbrev/create_new_list/{list_name}", createNewList)
	r.HandleFunc("/abbrev/delete_list/{list_name}", deleteList)
	r.HandleFunc("/abbrev/list_abbrevs/{list_name}", listAbbrevs)
	r.HandleFunc("/abbrev/add/{list_name}/{abbrev}/{expansion}", addAbbrev)
	r.HandleFunc("/abbrev/add_create_list_if_not_exists/{list_name}/{abbrev}/{expansion}", addAbbrevCreateListIfNotExists)
	r.HandleFunc("/abbrev/delete/{list_name}/{abbrev}", deleteAbbrev)

	r.HandleFunc("/reload_validation_config", reloadValidationConfig)

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(*cfg.StaticDir))))

	srv := &http.Server{
		Handler:      r,
		Addr:         *cfg.Host + ":" + *cfg.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Info("[main] Starting server on %s://%s", *cfg.Protocol, *cfg.Host+":"+*cfg.Port)
	log.Info("[main] Serving dir %s", *cfg.StaticDir)

	go func() {
		// wait for the server to start, and then load data, including URL access tests
		// (which won't work if it's run before the server is started)
		time.Sleep(1000)
		valRes, err := proj.LoadData()
		levelFreq := map[string]int{}
		for _, vr := range valRes {
			log.Warning("%s\t%s", vr.Level, vr.Message)
			levelFreq[vr.Level]++
		}
		if len(levelFreq) > 0 {
			log.Warning("\n\n===== NUMBER OF ISSUES ======\n")
		}
		for k, v := range levelFreq {
			log.Warning("%s\t%d", k, v)
		}

		if err != nil {
			log.Fatal("Couldn't load data: %v", err)
		}
	}()

	// pings each client websocket
	go keepAlive()

	if err = srv.ListenAndServe(); err != nil {
		log.Fatal("Server failure: %v", err)
	}

}
