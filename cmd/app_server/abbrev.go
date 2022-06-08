// NB! This is slightly edited version of https://github.com/stts-se/tord3000/blob/master/main.go

package main

import (
	"encoding/json"
	"fmt"
	//"io/ioutil"
	"net/http"

	//"os/exec"
	//"os/user"
	//"path/filepath"
	"sort"
	//"strings"
	"sync"
	//"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stts-se/transtool/abbrevs"
	"github.com/stts-se/transtool/log"
)

// === CONSTANTS
//const timestampFmt = "2006-01-02 15:04:05 CET" // time.UnixDate // "%Y-%m-%d %H:%M:%S" // "2019-11-04 15:34 CET"

// === FIELDS

//var baseDir = "."

// Initialised in main.go
var abbrevManager abbrevs.AbbrevManager

// type producer struct {
// 	userName  string
// 	session   string
// 	conn      *websocket.Conn
// 	timestamp time.Time
// }

// TODO: One Mutex for each
var sessionMutex sync.Mutex

//var wsConsumers = make(map[string]map[string]consumer) // sessionName => userName => connection
//var wsProducers = make(map[string]producer)            // sessionName => connection
//var wsConsumersGlobalListeners = []*websocket.Conn{} // listen to global changes (not session specific)
var wsProducersGlobalListeners = []*websocket.Conn{} // listen to global changes (not session specific)

//var upgrader = websocket.Upgrader{}

// This is filled in by main, listing the URLs handled by the router,
// so that these can be shown in the generated docs.
//var walkedURLs []string

// ==== UTILITIES

// for neater request param validation
// type param struct {
// 	name  string
// 	value string
// }

type SocketJSON struct {
	Label   string `json:"label"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// type consumer struct {
// 	userName string
// 	//session   string
// 	conn      *websocket.Conn
// 	timestamp time.Time
// }

// TODO Gorilla websocket has methods for handling ping and pong, but it must be set up at both ends.
// TODO *ALL* keepalive messages are sent within the same mutex lock. This may not be nice. Potentially, this freezes the server every 23rd second.
// TODO Either add mutexes for the individual maps/slices, or run in parallell go rutines, or use channels perhaps
// func keepWSAlive() {

// 	//TODO Maybe set individual mutexes for each map/slice?

// 	js := SocketJSON{Label: "keepalive"}
// 	jsb, err := json.Marshal(js)
// 	if err != nil {
// 		msg := fmt.Sprintf("keepWSAlive: failed to marshal %v : %v", js, err)
// 		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
// 		log.Info(msg)
// 		return
// 	}

// 	t := time.NewTicker(23 * time.Second)
// 	for _ = range t.C {

// 		//log.Info("Sending keepalive")

// 		sessionMutex.Lock()

// 		var tmp = wsProducersGlobalListeners
// 		for _, conn := range tmp {
// 			err := conn.WriteMessage(websocket.TextMessage, jsb)
// 			if err != nil {
// 				msg := fmt.Sprintf("keepWSAlive: failure to write to socket : %v", err)
// 				log.Info(msg)
// 				wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
// 			}

// 		}

// 		tmp = wsConsumersGlobalListeners
// 		for _, conn := range tmp {
// 			err = conn.WriteMessage(websocket.TextMessage, jsb)
// 			if err != nil {
// 				msg := fmt.Sprintf("keepWSAlive: failure to write to socket : %v", err)
// 				wsConsumersGlobalListeners = deleteConnection(conn, wsConsumersGlobalListeners)
// 				log.Info(msg)
// 			}

// 		}

// 		for _, conns := range wsConsumers {
// 			for cName, conn := range conns {
// 				err = conn.conn.WriteMessage(websocket.TextMessage, jsb)
// 				if err != nil {
// 					msg := fmt.Sprintf("keepWSAlive: failure to write to socket : %v", err)
// 					delete(conns, cName)
// 					log.Info(msg)
// 				}
// 			}
// 		}
// 		for cName, conn := range wsProducers {
// 			err = conn.conn.WriteMessage(websocket.TextMessage, jsb)
// 			if err != nil {
// 				msg := fmt.Sprintf("keepWSAlive: failure to write to socket : %v", err)
// 				delete(wsProducers, cName)
// 				log.Info(msg)
// 			}
// 		}

// 		sessionMutex.Unlock()
// 	}
// }

// func newParam(name string) param {
// 	return param{name: name, value: ""}
// }

// func requireParams(vars map[string]string, params ...*param) error {
// 	missing := []string{}
// 	for _, p := range params {
// 		v, ok := vars[p.name]
// 		if !ok || strings.TrimSpace(v) == "" {
// 			missing = append(missing, p.name)
// 		} else {
// 			p.value = v
// 		}
// 	}
// 	if len(missing) > 0 {
// 		paramString := "param"
// 		if len(missing) != 1 {
// 			paramString += "s"
// 		}
// 		return fmt.Errorf("missing required %s: %s", paramString, strings.Join(missing, ", "))
// 	}
// 	return nil
// }

// exec in a mutex lock context only
func deleteConnection(conn *websocket.Conn, conns []*websocket.Conn) []*websocket.Conn {
	res := []*websocket.Conn{}
	for _, c := range conns {
		if conn != c {
			res = append(res, c)
		}
	}
	log.Info("Deleted global listener (# before: %d, # after: %d)", len(conns), len(res))
	return res
}

// print serverMsg to server log, and return an http error with clientMsg and the specified error code (http.StatusInternalServerError, etc)
// func httpError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
// 	log.Info(serverMsg)
// 	http.Error(w, clientMsg, errCode)
// }

// func errorToWebsocket(conn *websocket.Conn, serverMsg string, clientMsg string) {
// 	sendJS := SocketJSON{
// 		Label:   "error",
// 		Content: clientMsg,
// 	}
// 	jsb, err := json.Marshal(sendJS)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 		log.Info(msg)
// 		// TODO: We have no reponsewriter to write to -- can we send the http error to the websocket? How?
// 		//httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}
// 	err = conn.WriteMessage(websocket.TextMessage, jsb)
// 	if err != nil {
// 		msg := fmt.Sprintf("errorToWebsocket: failure to write to socket : %v", err)
// 		log.Info(msg)
// 		return
// 	}
// 	log.Info(serverMsg)
// }

func errorToResponseWriter(w http.ResponseWriter, serverMsg string, clientMsg string) {
	jsMessageToResponseWriter(w, "error", serverMsg, clientMsg)
}

func infoToResponseWriter(w http.ResponseWriter, serverMsg string, clientMsg string) {
	jsMessageToResponseWriter(w, "info", serverMsg, clientMsg)
}

func jsMessageToResponseWriter(w http.ResponseWriter, label string, serverMsg string, clientMsg string) {
	sendJS := SocketJSON{
		Label:   label,
		Content: clientMsg,
	}
	jsb, err := json.Marshal(sendJS)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
		log.Info(msg)
		// TODO: We have no reponsewriter to write to -- can we send the http error to the websocket? How?
		//httpError(w, msg, msg, http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, string(jsb))
	log.Info(serverMsg)
}

// ==== ABBREVS

// Abbrev is a tuple holding an abbreviation and its expansion.
type Abbrev struct {
	Abbrev    string `json:"abbrev"`
	Expansion string `json:"expansion"`
}

func listLists(w http.ResponseWriter, r *http.Request) {
	//res := []string{}

	// TODO sort
	lists := abbrevManager.Lists()
	//Sort abbreviations alphabetically-ish
	sort.Slice(lists, func(i, j int) bool { return lists[i] < lists[j] })

	resJSON, err := json.Marshal(lists)
	if err != nil {
		msg := fmt.Sprintf("lists: failed to marshal list of strings : %v", err)
		httpError(w, msg, "failed to return list of abbreviation lists", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(resJSON))

}

// func listProducerSessions(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	res := []string{}

// 	for s := range wsProducers {
// 		res = append(res, s)
// 	}

// 	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })

// 	resJSON, err := json.Marshal(res)
// 	if err != nil {
// 		msg := fmt.Sprintf("listProducerSessions: failed to marshal list of strings : %v", err)
// 		httpError(w, msg, "failed to return list of producer sessions", http.StatusInternalServerError)
// 		return
// 	}

// 	fmt.Fprint(w, string(resJSON))

// }

func listListsWithLength(w http.ResponseWriter, r *http.Request) {
	//res := []string{}

	// TODO sort
	lists := abbrevManager.ListsWithLength()
	//Sort abbreviations alphabetically-ish
	sort.Slice(lists, func(i, j int) bool { return lists[i].Name < lists[j].Name })

	resJSON, err := json.Marshal(lists)
	if err != nil {
		msg := fmt.Sprintf("lists: failed to marshal list of strings : %v", err)
		httpError(w, msg, "failed to return list of abbreviation lists", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(resJSON))

}

func createNewList(w http.ResponseWriter, r *http.Request) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	params := mux.Vars(r)
	listName := params["list_name"]
	err := abbrevManager.CreateList(listName)
	if err != nil {
		msg := fmt.Sprintf("failed to create new abbreviation list : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}

	js := SocketJSON{
		Label: "abbrev_lists_updated",
	}
	jsb, err := json.Marshal(js)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
		log.Info(msg)
		return
	}

	var tmp = wsProducersGlobalListeners
	for _, conn := range tmp {
		err := conn.WriteMessage(websocket.TextMessage, jsb)
		if err != nil {
			msg := fmt.Sprintf("createNewList: failure to write to socket : %v", err)
			log.Info(msg)
			wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
		}
	}

}

func deleteList(w http.ResponseWriter, r *http.Request) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	params := mux.Vars(r)
	listName := params["list_name"]
	err := abbrevManager.DeleteListFile(listName)
	if err != nil {
		msg := fmt.Sprintf("failed to delete list : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	js := SocketJSON{
		Label: "abbrev_lists_updated",
	}
	jsb, err := json.Marshal(js)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
		log.Info(msg)
		return
	}

	var tmp = wsProducersGlobalListeners
	for _, conn := range tmp {
		err := conn.WriteMessage(websocket.TextMessage, jsb)
		if err != nil {
			msg := fmt.Sprintf("deleteList: failure to write to socket : %v", err)
			log.Info(msg)
			wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
		}

	}

}

func listAbbrevs(w http.ResponseWriter, r *http.Request) {
	res := []Abbrev{}

	params := mux.Vars(r)
	listName := params["list_name"]

	//abbrevMutex.RLock()
	//defer abbrevMutex.RUnlock()

	for k, v := range abbrevManager.AbbrevsFor(listName) {
		res = append(res, Abbrev{Abbrev: k, Expansion: v})
	}

	//Sort abbreviations alphabetically-ish
	sort.Slice(res, func(i, j int) bool { return res[i].Abbrev < res[j].Abbrev })

	resJSON, err := json.Marshal(res)
	if err != nil {
		msg := fmt.Sprintf("listAbbrevs: failed to marshal map of abbreviations : %v", err)
		httpError(w, msg, "failed to return list of abbreviations", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(resJSON))

}

func addAbbrev(w http.ResponseWriter, r *http.Request) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	params := mux.Vars(r)
	listName := params["list_name"]
	abbrev := params["abbrev"]
	expansion := params["expansion"]

	// TODO Error check that abbrev doesn't already exist in map
	//	abbrevMutex.Lock()
	//	abbrevs[abbrev] = expansion
	//	abbrevMutex.Unlock() // Can't use defer here, since call below uses
	// locking

	// This could be done consurrently, but easier to catch errors this way
	err := abbrevManager.Add(listName, abbrev, expansion)
	if err != nil {
		msg := fmt.Sprintf("failed to save abbrev : %v", err)
		//httpError(w, msg, "failed to save abbreviation", http.StatusInternalServerError)
		errorToResponseWriter(w, msg, msg)
		return
	}
	js := SocketJSON{
		Label: "abbrevs_updated",
	}
	jsb, err := json.Marshal(js)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
		log.Info(msg)
		return
	}

	var tmp = wsProducersGlobalListeners
	for _, conn := range tmp {
		err := conn.WriteMessage(websocket.TextMessage, jsb)
		if err != nil {
			msg := fmt.Sprintf("addAbbrev: failure to write to socket : %v", err)
			log.Info(msg)
			wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
		}

	}

	msg := fmt.Sprintf("saved abbreviation '%s' '%s' '%s'", listName, abbrev, expansion)
	infoToResponseWriter(w, msg, msg)
}

func addAbbrevCreateListIfNotExists(w http.ResponseWriter, r *http.Request) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	params := mux.Vars(r)
	listName := params["list_name"]
	abbrev := params["abbrev"]
	expansion := params["expansion"]

	// TODO Error check that abbrev doesn't already exist in map
	//	abbrevMutex.Lock()
	//	abbrevs[abbrev] = expansion
	//	abbrevMutex.Unlock() // Can't use defer here, since call below uses
	// locking

	// This could be done consurrently, but easier to catch errors this way
	err := abbrevManager.AddCreateIfNotExists(listName, abbrev, expansion)
	if err != nil {
		msg := fmt.Sprintf("failed to save abbrev : %v", err)
		//httpError(w, msg, "failed to save abbreviation", http.StatusInternalServerError)
		errorToResponseWriter(w, msg, msg)
		return
	}
	js := SocketJSON{
		Label: "abbrevs_updated",
	}
	jsb, err := json.Marshal(js)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
		log.Info(msg)
		return
	}

	var tmp = wsProducersGlobalListeners
	for _, conn := range tmp {
		err := conn.WriteMessage(websocket.TextMessage, jsb)
		if err != nil {
			msg := fmt.Sprintf("addAbbrevCreateListIfNotExists: failure to write to socket : %v", err)
			log.Info(msg)
			wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
		}

	}

	msg := fmt.Sprintf("saved abbreviation '%s' '%s' '%s'", listName, abbrev, expansion)
	infoToResponseWriter(w, msg, msg)
}

func deleteAbbrev(w http.ResponseWriter, r *http.Request) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	params := mux.Vars(r)
	listName := params["list_name"]
	abbrev := params["abbrev"]
	//expansion := params["expansion"]

	// TODO Error check that abbrev doesn't already exist in map
	//abbrevMutex.Lock()
	//delete(abbrevs, abbrev)
	//abbrevMutex.Unlock() // Can't use defer here, since call below uses
	// locking

	// This could be done concurrently, but easier to catch errors this way
	err := abbrevManager.Delete(listName, abbrev)
	if err != nil {
		msg := fmt.Sprintf("deleteAbbrev: failed to delete abbrev : %v", err)
		httpError(w, msg, "failed to delete abbreviation(s)", http.StatusInternalServerError)
		return
	}
	js := SocketJSON{
		Label: "abbrevs_updated",
	}
	jsb, err := json.Marshal(js)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
		log.Info(msg)
		return
	}

	var tmp = wsProducersGlobalListeners
	for _, conn := range tmp {
		err := conn.WriteMessage(websocket.TextMessage, jsb)
		if err != nil {
			msg := fmt.Sprintf("deleteAbbrev: failure to write to socket : %v", err)
			log.Info(msg)
			wsProducersGlobalListeners = deleteConnection(conn, wsProducersGlobalListeners)
		}

	}

	fmt.Fprintf(w, "deleted abbreviation '%s' '%s'\n", listName, abbrev)
}

// func producerWebsocketGlobalListener(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to upgrade to ws: %v", err)
// 		httpError(w, msg, "Failed to upgrade to ws", http.StatusBadRequest)
// 		return
// 	}
// 	wsProducersGlobalListeners = append(wsProducersGlobalListeners, conn)
// 	log.Info("Added global producer listener")
// }

// func consumerWebsocketGlobalListener(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to upgrade to ws: %v", err)
// 		httpError(w, msg, "Failed to upgrade to ws", http.StatusBadRequest)
// 		return
// 	}
// 	wsConsumersGlobalListeners = append(wsConsumersGlobalListeners, conn)
// 	log.Info("Added global consumer listener")
// }

// This is when the socket is created from the client
// func producerWebSocket(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	var session = newParam("session")
// 	var user = newParam("user")
// 	vars := mux.Vars(r)
// 	err := requireParams(vars, &session, &user)
// 	if err != nil {
// 		msg := "Invalid input params"
// 		httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to upgrade to ws: %v", err)
// 		httpError(w, msg, "Failed to upgrade to ws", http.StatusBadRequest)
// 		return
// 	}

// 	if _, ok := wsProducers[session.value]; ok {
// 		msg := fmt.Sprintf("Session already exists: %s", session.value)
// 		errorToWebsocket(conn, msg, "producerWebsocket: "+msg)
// 		//TODO httpError(w, msg, msg, http.StatusInternalServerError)
// 		return
// 	}

// 	log.Info("Producer websocket %s %s", session, user)
// 	prod := producer{
// 		userName:  user.value,
// 		session:   session.value,
// 		conn:      conn,
// 		timestamp: time.Now(),
// 	}
// 	wsProducers[session.value] = prod
// 	go broadCastProducer(prod)
// 	js := SocketJSON{
// 		Label:   "registered",
// 		Content: fmt.Sprintf("producer %s/%s", user.value, session.value),
// 	}
// 	jsb, err := json.Marshal(js)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
// 		//errorToWebsocket(conn, msg, "producerWebsocket: "+msg)
// 		//httpError(w, msg, "json marshal failed", http.StatusInternalServerError)
// 		log.Info(msg)
// 		return
// 	}
// 	err = conn.WriteMessage(websocket.TextMessage, jsb)
// 	if err != nil {
// 		msg := fmt.Sprintf("producerWebsocket: failure to write to socket : %v", err)
// 		//httpError(w, msg, msg, http.StatusBadRequest)
// 		//errorToWebsocket(conn, msg, "producerWebsocket: "+msg)
// 		log.Info(msg)
// 		return
// 	}
// 	broadcastProducerList()

// }

// func broadCastProducer(prod producer) {
// 	defer prod.conn.Close() // ??

// 	for {
// 		var js SocketJSON
// 		err := prod.conn.ReadJSON(&js)
// 		if err != nil {
// 			log.Info("broadCastProducer: failed to read websocket : %v", err)
// 			delete(wsProducers, prod.session)
// 			log.Info("deleted producer session %s", prod.session)
// 			sessionMutex.Lock()
// 			broadcastProducerList()
// 			sessionMutex.Unlock()
// 			return

// 		}
// 		if js.Label == "text" {
// 			sessionMutex.Lock()
// 			if cConns, ok := wsConsumers[prod.session]; ok {
// 				jsb, err := json.Marshal(js)
// 				if err != nil {
// 					msg := fmt.Sprintf("failed to marshal %v : %v", js, err)
// 					//httpError(prod.conn, msg, "json marshal failed", http.StatusInternalServerError)
// 					log.Info(msg)
// 					sessionMutex.Unlock()
// 					return
// 				}

// 				for userName, c := range cConns {
// 					err := c.conn.WriteMessage(websocket.TextMessage, jsb)
// 					if err != nil {
// 						log.Info("failed to write to ws : %v", err)
// 						delete(cConns, userName)
// 						wsConsumers[prod.session] = cConns
// 						broadcastConsumerList(prod)
// 					}
// 					//log.Info("wrote message to ws: '%s'", js.Content)
// 				}
// 			}
// 		}
// 		sessionMutex.Unlock()
// 	}
// }

// func broadcastProducerList() {
// 	// sessionMutex.Lock()
// 	// defer sessionMutex.Unlock()

// 	prodSessions := []string{}
// 	for _, prod := range wsProducers {
// 		prodSessions = append(prodSessions, prod.session)
// 	}
// 	sendJS := SocketJSON{
// 		Label:   "registerer_producers",
// 		Content: strings.Join(prodSessions, ", "),
// 	}
// 	jsb, err := json.Marshal(sendJS)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 		log.Info(msg)
// 		//panic(msg)
// 		return
// 	}

// 	var tmp = wsConsumersGlobalListeners
// 	for _, conn := range tmp {
// 		err = conn.WriteMessage(websocket.TextMessage, jsb)
// 		if err != nil {
// 			msg := fmt.Sprintf("broadcastProducerList: failure to write to socket : %v", err)
// 			wsConsumersGlobalListeners = deleteConnection(conn, wsConsumersGlobalListeners)
// 			log.Info(msg)
// 		}

// 	}
// }

// func broadcastConsumerList(prod producer) {
// 	// sessionMutex.Lock()
// 	// defer sessionMutex.Unlock()

// 	if cConns, ok := wsConsumers[prod.session]; ok {
// 		consumers := []string{}
// 		for _, cons := range cConns {
// 			consumers = append(consumers, fmt.Sprintf("%s; %v", cons.userName, cons.timestamp.Format(timestampFmt)))
// 		}
// 		sendJS := SocketJSON{
// 			Label:   "registered_consumers",
// 			Content: strings.Join(consumers, ", "),
// 		}
// 		jsb, err := json.Marshal(sendJS)
// 		if err != nil {
// 			msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 			log.Info(msg)
// 			//panic(msg)
// 			return
// 		}
// 		err = prod.conn.WriteMessage(websocket.TextMessage, jsb)
// 		if err != nil {
// 			msg := fmt.Sprintf("broadcastConsumerList: failure to write to socket : %v", err)
// 			//httpError(w, msg, msg, http.StatusBadRequest) // TODO: We have no reponsewriter to write to -- can we send the http error to the websocket? How?
// 			log.Info(msg)
// 			return
// 		}

// 	}
// }

// func unregisterConsumer(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	var session = newParam("session")
// 	var user = newParam("user")
// 	vars := mux.Vars(r)
// 	err := requireParams(vars, &session, &user)
// 	if err != nil {
// 		msg := "Invalid input params"
// 		httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}
// 	deleted := false
// 	if conns, ok := wsConsumers[session.value]; ok {
// 		if _, ok := conns[user.value]; ok {
// 			log.Info("Unregister consumer %s %s", session, user)
// 			delete(conns, user.value)
// 			deleted = true
// 		}
// 	}
// 	if !deleted {
// 		msg := fmt.Sprintf("No consumer session for %s/%s", session.value, user.value)
// 		httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}

// 	// broadcast unregister to producer
// 	if prod, ok := wsProducers[session.value]; deleted && ok {
// 		sendJS := SocketJSON{
// 			Label:   "unregistered",
// 			Content: fmt.Sprintf("consumer %s/%s", user.value, session.value),
// 		}
// 		jsb, err := json.Marshal(sendJS)
// 		if err != nil {
// 			msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 			log.Info(msg)
// 			//panic(msg)
// 			return
// 		}
// 		err = prod.conn.WriteMessage(websocket.TextMessage, jsb)
// 		if err != nil {
// 			msg := fmt.Sprintf("unregisterConsumer: failure to write to socket : %v", err)
// 			//httpError(w, msg, msg, http.StatusBadRequest)
// 			log.Info(msg)
// 			return
// 		}

// 		broadcastConsumerList(prod)
// 	}
// 	//return
// }

// func unregisterProducer(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	var session = newParam("session")
// 	var user = newParam("user")
// 	vars := mux.Vars(r)
// 	err := requireParams(vars, &session, &user)
// 	if err != nil {
// 		msg := "Invalid input params"
// 		httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}
// 	log.Info("Unregister producer %s %s", session, user)
// 	prod, ok := wsProducers[session.value]
// 	if ok {
// 		delete(wsProducers, session.value)
// 	}

// 	// broadcast unregister to consumers
// 	if cConns, ok := wsConsumers[session.value]; ok {

// 		sendJS := SocketJSON{
// 			Label:   "unregistered",
// 			Content: fmt.Sprintf("producer %s/%s", user.value, session.value),
// 		}
// 		jsb, err := json.Marshal(sendJS)
// 		if err != nil {
// 			msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 			log.Info(msg)
// 			//panic(msg)
// 			return
// 		}

// 		for userName, c := range cConns {
// 			err := c.conn.WriteMessage(websocket.TextMessage, jsb)
// 			if err != nil {
// 				log.Info("failed to write to ws : %v", err)
// 				delete(cConns, userName)
// 				wsConsumers[session.value] = cConns
// 				broadcastConsumerList(prod)
// 			}
// 			//log.Info("wrote message to ws: '%s'", js.Content)
// 		}
// 	}
// 	broadcastProducerList()
// }

// func consumerWebSocket(w http.ResponseWriter, r *http.Request) {
// 	sessionMutex.Lock()
// 	defer sessionMutex.Unlock()

// 	var session = newParam("session")
// 	var user = newParam("user")
// 	vars := mux.Vars(r)
// 	err := requireParams(vars, &session, &user)
// 	if err != nil {
// 		msg := "Invalid input params"
// 		httpError(w, msg, msg, http.StatusBadRequest)
// 		return
// 	}
// 	log.Info("Consumer websocket %s %s", session, user)
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		msg := fmt.Sprintf("failed to upgrade to ws: %v", err)
// 		httpError(w, msg, "Failed to upgrade to ws", http.StatusBadRequest)
// 		return
// 	}
// 	if prod, ok := wsProducers[session.value]; ok {
// 		if _, ok := wsConsumers[session.value]; !ok {
// 			wsConsumers[session.value] = make(map[string]consumer)
// 		}
// 		if _, ok := wsConsumers[session.value][user.value]; ok {
// 			msg := fmt.Sprintf("User %s is already subscribed to session %s", user.value, session.value)
// 			errorToWebsocket(conn, msg, "consumerWebSocket: "+msg)
// 			return
// 		}

// 		cons := consumer{
// 			userName:  user.value,
// 			conn:      conn,
// 			timestamp: time.Now(),
// 		}
// 		wsConsumers[session.value][user.value] = cons
// 		sendJS := SocketJSON{
// 			Label:   "subscribed",
// 			Content: fmt.Sprintf("consumer %s/%s", user.value, session.value),
// 		}
// 		jsb, err := json.Marshal(sendJS)
// 		if err != nil {
// 			msg := fmt.Sprintf("failed to marshal %v : %v", sendJS, err)
// 			//httpError(w, msg, msg, http.StatusBadRequest)
// 			errorToWebsocket(conn, msg, "consumerWebSocket: "+msg)
// 			//panic(msg)
// 			return
// 		}
// 		err = prod.conn.WriteMessage(websocket.TextMessage, jsb)
// 		if err != nil {
// 			msg := fmt.Sprintf("consumerWebSocket: failure to write to socket : %v", err)
// 			//httpError(w, msg, msg, http.StatusBadRequest)
// 			//errorToWebsocket(conn, msg, "consumerWebSocket: "+msg)
// 			log.Info(msg)
// 			return
// 		}
// 		err = conn.WriteMessage(websocket.TextMessage, jsb)
// 		if err != nil {
// 			msg := fmt.Sprintf("consumerWebSocket: failure to write to socket : %v", err)
// 			//httpError(w, msg, msg, http.StatusBadRequest)
// 			//errorToWebsocket(conn, msg, "consumerWebSocket: "+msg)
// 			log.Info(msg)
// 			return
// 		}

// 		broadcastConsumerList(prod)
// 	} else {
// 		msg := "No producer session for " + session.value
// 		errorToWebsocket(conn, msg, "consumerWebSocket: "+msg)
// 		return
// 	}

// }

// TODO Use a HTML template to generate complete page?
// func generateDoc(w http.ResponseWriter, r *http.Request) {
// 	//sort.Slice(walkedURLs, func(i, j int) bool { return walkedURLs[i] < walkedURLs[j] })
// 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
// 	fmt.Fprintf(w, "<html><head><title>%s</title></head><body>", "T3000: Doc")
// 	for _, url := range walkedURLs {
// 		fmt.Fprintf(w, "%s<br/>\n", url)
// 	}
// 	fmt.Fprintf(w, "</body></html>")
// }

// func getBuildInfo(prefix string, lines []string, defaultValue string) []string {
// 	for _, l := range lines {
// 		fs := strings.Split(l, ": ")
// 		if fs[0] == prefix {
// 			return fs
// 		}
// 	}
// 	return []string{prefix, defaultValue}
// }

//const buildInfoFile = "buildinfo.txt"

// func generateAbout(w http.ResponseWriter, r *http.Request) {

// 	bytes, err := ioutil.ReadFile(filepath.Clean(buildInfoFile))
// 	if err != nil {
// 		log.Info("failed loading file : %v", err)
// 	}
// 	buildInfoLines := strings.Split(strings.TrimSpace(string(bytes)), "\n")

// 	res := [][]string{}
// 	res = append(res, []string{"Application name", "T3000"})

// 	// build timestamp
// 	res = append(res, getBuildInfo("Build timestamp", buildInfoLines, "n/a"))
// 	user, err := user.Current()
// 	if err != nil {
// 		log.Info("failed reading system user name : %v", err)
// 	}

// 	// built by username
// 	res = append(res, getBuildInfo("Built by", buildInfoLines, user.Name))

// 	// git commit id and branch
// 	commitIDLong, err := exec.Command("git", "rev-parse", "HEAD").Output()
// 	var commitIDAndBranch = "unknown"
// 	if err != nil {
// 		log.Info("couldn't retrieve git commit hash: %v", err)
// 	} else {
// 		commitID := string([]rune(string(commitIDLong)[0:7]))
// 		branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
// 		if err != nil {
// 			log.Info("couldn't retrieve git branch: %v", err)
// 		} else {
// 			commitIDAndBranch = fmt.Sprintf("%s on %s", commitID, strings.TrimSpace(string(branch)))
// 		}
// 	}
// 	res = append(res, getBuildInfo("Git commit", buildInfoLines, commitIDAndBranch))

// 	// git release tag
// 	releaseTag, err := exec.Command("git", "describe", "--tags").Output()
// 	if err != nil {
// 		log.Info("couldn't retrieve git release/tag: %v", err)
// 		releaseTag = []byte("unknown")
// 	}
// 	res = append(res, getBuildInfo("Release", buildInfoLines, string(releaseTag)))

// 	//res = append(res, []string{"Started", start.Format(timestampFmt)})
// 	//res = append(res, []string{"Host", host})
// 	//res = append(res, []string{"Port", port})
// 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
// 	fmt.Fprintf(w, "<html><head><title>%s</title></head><body>", "T3000: About")
// 	fmt.Fprintf(w, "<table><tbody>")
// 	for _, l := range res {
// 		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td></tr>\n", l[0], l[1])
// 	}
// 	fmt.Fprintf(w, "</tbody></table>")
// 	fmt.Fprintf(w, "</body></html>")
// }
