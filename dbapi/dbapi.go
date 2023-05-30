package dbapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	// "golang.org/x/exp/maps"
	// "golang.org/x/exp/slices"

	"github.com/stts-se/transtool/log"
	"github.com/stts-se/transtool/protocol"
	"github.com/stts-se/transtool/validation"
)

const (
	debug = false
)

type ClientID struct {
	ID       string `json:"id"`
	UserName string `json:"user_name"`
}

// Proj is a struct wrapping a set of dbapi.DBAPI:s
type Proj struct {
	mutex         *sync.RWMutex
	DBs           map[string]*DBAPI
	statusSources map[string]bool // To keep track of user names
	validator     *validation.Validator
}

// NewProj takes a colon separated list of project sub directories
func NewProj(dirList string, validator *validation.Validator) (Proj, error) {
	res := Proj{
		mutex:         &sync.RWMutex{},
		DBs:           map[string]*DBAPI{},
		statusSources: map[string]bool{},
	}

	paths := strings.Split(dirList, ":")

	for _, p := range paths {
		p = strings.TrimSpace(p)

		// remove potential trailing slash from dir name, in
		// order to avoid later mismatch, since there are
		// currently used as the name of the sub proj
		// (in Proj.DBs:   map[string]*DBAPI{})
		p = strings.TrimSuffix(p, "/")

		_, err := os.Stat(p)
		if os.IsNotExist(err) {
			return res, fmt.Errorf("non-existing directory : %v", err)

		}

		sources := path.Join(p, "source")
		_, err = os.Stat(sources)
		if os.IsNotExist(err) {
			return res, fmt.Errorf("non-existing sources directory : %v", err)
		}

		annotation := path.Join(p, "annotation")
		_, err = os.Stat(annotation)
		if os.IsNotExist(err) {
			return res, fmt.Errorf("non-existing annotation directory : %v", err)
		}

		db := NewDBAPI(p, validator)
		if _, ok := res.DBs[p]; ok {
			fmt.Fprintf(os.Stderr, "directory already loaded, skipping: '%s'\n", p)
			continue
			//return res, fmt.Errorf("directory already loaded: '%s'", p)
		}
		res.DBs[p] = db

	}

	return res, nil
}

// GetStatusSources returns a list of the "status sources" (typically editor user names) known in the project
func (p *Proj) GetStatusSources() []string {
	var res []string
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for s := range p.statusSources {
		res = append(res, s)
	}

	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })

	return res
}

// TODO Return error if no DBAPI of name projName?

// TODO Should it be DBAPI rather than *DBAPI? The pointer version is
// kept from earlier version, without Proj struct

func (p *Proj) GetDB(projName string) *DBAPI {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.DBs[projName]
}

func (p *Proj) ListSubProjs() []string {
	var res []string
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for sp := range p.DBs {
		res = append(res, sp)
	}

	return res
}

func (p *Proj) Stats() map[string]SubProjStats {
	res := map[string]SubProjStats{}

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for name, db := range p.DBs {
		stats := db.StatsII()
		res[name] = stats
	}
	return res
}

func (p *Proj) UserIDsForLockedPages() []ClientID {
	var res []ClientID
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for _, db := range p.DBs {
		ids := db.userIDsForLockedPages()
		res = append(res, ids...)
	}

	return res
}

// Unlock wraps dbapi.DBAPI.Unlock
func (p *Proj) Unlock(subProj, pageID string, ci ClientID) error {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[subProj]
	p.mutex.RUnlock()
	if !ok {
		return fmt.Errorf("dbapi.Proj.Unlock: unknown subProj '%s'", subProj)
	}

	return db.Unlock(pageID, ci)
}

// UnlockAll wraps dbapi.DBAPI.Unlock
func (p *Proj) UnlockAll(ci ClientID) (int, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Currently ignores subProj and calls db.UnlockAll on every subProj.
	// The reason is that if the client is in a confused state, the subProj may be incorrect
	// (this shouldn't happen, but in case)

	res := 0
	for _, db := range p.DBs {
		n, err := db.UnlockAll(ci)
		if err != nil {
			return res, fmt.Errorf("error from dbapi.Proj.Unlock: '%v'", err)
		}
		res += n
	}

	return res, nil
}

type ValRes struct {
	Level   string
	Message string
}

// LoadData wraps dbapi.DBAPI.LoadData
func (p *Proj) LoadData() ([]ValRes, error) {
	var res []ValRes
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for sp, db := range p.DBs {
		vRes, err := db.LoadData()
		res = append(res, vRes...)
		if err != nil {

			msg := fmt.Sprintf("failed to load subproj '%s' : %v", sp, err)
			res = append(res, ValRes{"error", msg})
			log.Info("[dbapi] Failed to load data for '%s'", sp)
			continue
			//return res, fmt.Errorf("failed to load subproj '%s' : %v", sp, err)
		}
		log.Info("[dbapi] Loaded data for '%s'", sp)

		// collect "status sources", typically editor names
		for _, a := range db.annotationData {

			s := a.CurrentStatus.Source
			if s != "" {
				p.statusSources[s] = true
			}
			for _, c := range a.Chunks {
				if c.CurrentStatus.Source != "" {
					p.statusSources[c.CurrentStatus.Source] = true
				}
			}
		}

	}
	return res, nil
}

func (p *Proj) ListAudioFiles(subProj string) ([]string, error) {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[subProj]
	p.mutex.RUnlock()

	if !ok {
		return []string{}, fmt.Errorf("dbap.Proj.ListAudioFiles: no such sub project '%s'", subProj)
	}

	return db.ListAudioFiles(), nil
}

func (p *Proj) PageFromID(subProj, id string) (protocol.PagePayload, error) {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[subProj]
	p.mutex.RUnlock()
	if !ok {
		return protocol.PagePayload{}, fmt.Errorf("dbapi.Proj.PageFromID: no such sub proj '%s'", subProj)
	}

	return db.PageFromID(id)
}

func (p *Proj) BuildAudioPath(subProj, audioFile string) (string, error) {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[subProj]
	p.mutex.RUnlock()
	if !ok {
		return "", fmt.Errorf("dbapi.Proj.BuildAudioPath: no such sub proj '%s'", subProj)
	}

	return db.BuildAudioPath(audioFile)
}

func (p *Proj) Save(annotation protocol.AnnotationPayload) error {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[annotation.SubProj]
	p.mutex.RUnlock()
	if !ok {
		return fmt.Errorf("dbapi.Proj.Save: no such sub proj '%s'", annotation.SubProj)
	}

	// Test that annotation actually exist in sub proj db, not
	// to wrongly put a new file into dir.

	if _, ok := db.annotationData[annotation.Page.ID]; !ok {
		inOtherDB := ""
		for name, d := range p.DBs {
			if _, ok := d.annotationData[annotation.Page.ID]; ok {
				inOtherDB = name
				break
			}
		}

		msg := fmt.Sprintf("dbapi.Proj.Save: failed to save annotation, since annotation '%s' doesn't exist in sub proj '%s'", annotation.Page.ID, annotation.SubProj)

		if inOtherDB != "" {
			msg = fmt.Sprintf("%s. Annotation does exist in another sub proj, '%s'", msg, inOtherDB)
		}

		return fmt.Errorf(msg)
	}

	// Add editor names (so that all new names are in list)
	p.mutex.Lock()
	if annotation.CurrentStatus.Source != "" {
		p.statusSources[annotation.CurrentStatus.Source] = true

	}
	for _, c := range annotation.Chunks {
		if c.CurrentStatus.Source != "" {
			p.statusSources[c.CurrentStatus.Source] = true
		}
	}
	p.mutex.Unlock()

	return db.Save(annotation)
}

func (p *Proj) GetNextPage(subProj string, query protocol.QueryPayload, currentlyLockedID string, clientID ClientID, lockOnLoad bool) (protocol.AnnotationPayload, string, error) {
	p.mutex.RLock()
	//defer p.mutex.RUnlock()
	db, ok := p.DBs[subProj]
	p.mutex.RUnlock()
	if !ok {
		return protocol.AnnotationPayload{}, "", fmt.Errorf("dbapi.Proj.GetNextPage: no such sub proj '%s'", subProj)
	}

	a, s, e := db.GetNextPage(query, currentlyLockedID, clientID, lockOnLoad)
	//NL 20210715 TODO get SubProj to func db.GetNextPage in a proper way, so it can be returned
	a.SubProj = subProj
	return a, s, e
}

// TODO: only keep last two part of path as name, and validate that it is unique

type DBAPI struct {
	ProjectDir, SourceDataDir, AnnotationDataDir string

	dbMutex        *sync.RWMutex // for db read/write (files and in-memory saves)
	sourceData     []protocol.PagePayload
	annotationData map[string]protocol.AnnotationPayload

	lockMapMutex *sync.RWMutex       // for page locking
	lockMap      map[string]ClientID // page id -> user

	validator *validation.Validator
}

func NewDBAPI(projectDir string, validator *validation.Validator) *DBAPI {
	res := DBAPI{
		ProjectDir:        projectDir,
		SourceDataDir:     path.Join(projectDir, "source"),
		AnnotationDataDir: path.Join(projectDir, "annotation"),

		dbMutex:        &sync.RWMutex{},
		sourceData:     []protocol.PagePayload{},
		annotationData: map[string]protocol.AnnotationPayload{},

		lockMapMutex: &sync.RWMutex{},
		lockMap:      map[string]ClientID{},

		validator: validator,
	}
	return &res
}

func (api *DBAPI) ProjectName() string {
	return path.Base(api.ProjectDir)
}

func (api *DBAPI) PageFromID(id string) (protocol.PagePayload, error) {
	for _, page := range api.sourceData {
		if page.ID == id {
			// if anno, annotated := api.annotationData[page.ID]; annotated {
			// 	return anno.PagePayload, nil
			// }
			return page, nil
		}
	}
	return protocol.PagePayload{}, fmt.Errorf("no page with id: %s", id)
}

func (api *DBAPI) LoadData() ([]ValRes, error) {
	var res []ValRes
	var err error

	if api.ProjectDir == "" {
		return res, fmt.Errorf("project dir not provided")
	}
	if api.SourceDataDir == "" {
		return res, fmt.Errorf("source dir not set")
	}
	if api.AnnotationDataDir == "" {
		return res, fmt.Errorf("annotation dir not set")
	}

	info, err := os.Stat(api.ProjectDir)
	if os.IsNotExist(err) {
		return res, fmt.Errorf("project dir does not exist: %s", api.ProjectDir)
	}
	if !info.IsDir() {
		return res, fmt.Errorf("project dir is not a directory: %s", api.ProjectDir)
	}

	info, err = os.Stat(api.SourceDataDir)
	if os.IsNotExist(err) {
		return res, fmt.Errorf("source dir does not exist: %s", api.SourceDataDir)
	}
	if !info.IsDir() {
		return res, fmt.Errorf("source dir is not a directory: %s", api.SourceDataDir)
	}

	info, err = os.Stat(api.AnnotationDataDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(api.AnnotationDataDir, 0700)
		if err != nil {
			return res, fmt.Errorf("failed to create annotation folder %s : %v", api.AnnotationDataDir, err)
		}
		log.Info("[dbapi] Created annotation dir %s", api.AnnotationDataDir)
	} else if !info.IsDir() {
		return res, fmt.Errorf("annotation dir is not a directory: %s", api.AnnotationDataDir)
	}

	api.dbMutex.Lock()
	defer api.dbMutex.Unlock()
	var vRes []ValRes
	api.sourceData, vRes, err = api.LoadSourceData()
	res = append(res, vRes...)
	if err != nil {
		return res, fmt.Errorf("LoadSourceData() returned error : %v", err)
	}
	log.Info("[dbapi] Loaded %d source pages", len(api.sourceData))

	api.annotationData, vRes, err = api.LoadAnnotationData()
	res = append(res, vRes...)
	if err != nil {
		return res, fmt.Errorf("LoadAnnotationData() returned error : %v", err)
	}
	log.Info("[dbapi] Loaded %d annotation files", len(api.annotationData))

	vRes = api.validateData()
	res = append(res, vRes...)

	//log.Info("[dbapi] Data validated without errors")

	return res, err
}

func (api *DBAPI) listJSONFiles(dir string) []string {
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

func (api *DBAPI) validateData() []ValRes {
	var valRes []ValRes

	if len(api.sourceData) == 0 {
		//return fmt.Errorf("found no pages in source data")
		valRes = append(valRes, ValRes{"error", "found no pages in source data"})
	}
	if len(api.annotationData) == 0 {
		valRes = append(valRes, ValRes{"error", "found no annotation data"})
	}
	sourceMap := map[string]protocol.PagePayload{}
	for _, p := range api.sourceData {
		sourceMap[p.ID] = p
	}
	for id, anno := range api.annotationData {
		page, pageExists := sourceMap[id]
		if !pageExists {
			msg := fmt.Sprintf("annotation data with id %s not found in source data", id)
			valRes = append(valRes, ValRes{"error", msg})
		}
		if anno.Page.Audio != page.Audio {
			msg := fmt.Sprintf("annotation data has a different audio than source data: %s vs %s", anno.Page.Audio, page.Audio)
			valRes = append(valRes, ValRes{"error", msg})
		}
		if anno.Page.ID != page.ID {
			msg := fmt.Sprintf("annotation data has a different ID than source data: %s vs %s", anno.Page.ID, page.ID)
			valRes = append(valRes, ValRes{"error", msg})
		}
	}
	a := []protocol.PagePayload{}
	for _, page := range api.sourceData {
		_, annoExists := api.annotationData[page.ID]
		if !annoExists {
			//msg := fmt.Sprintf("page data with id %s not found in annotation data", page.ID)
			msg := fmt.Sprintf("page data with id %s not found in annotation data (removing the page)", page.ID)
			valRes = append(valRes, ValRes{"error", msg})
		} else {
			a = append(a, page)
		}
	}
	api.sourceData = a
	return valRes
}

func testAudioAccess(buildAudio func(string) string, page protocol.PagePayload) error {
	urlResp, err := http.Get(buildAudio(page.Audio))
	if err != nil {
		return fmt.Errorf("audio Audio %s not reachable : %v", page.Audio, err)
	}
	defer urlResp.Body.Close()
	if urlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("audio Audio %s not reachable (status %s)", page.Audio, urlResp.Status)
	}
	return nil
}

func (api *DBAPI) BuildAudioPath(audioFile string) (string, error) {
	if audioFile == "" {
		return "", fmt.Errorf("empty audio path")
	}
	return path.Join(api.SourceDataDir, audioFile), nil
}

func (api *DBAPI) TestAudioAccess(buildAudio func(string) string) error {
	for _, page := range api.sourceData {
		if err := testAudioAccess(buildAudio, page); err != nil {
			return err
		}
	}
	return nil
}

func (api *DBAPI) validatePage(page protocol.PagePayload) error {
	if page.ID == "" {
		return fmt.Errorf("no id")
	}
	if page.Audio == "" {
		return fmt.Errorf("no audio")
	}
	if page.Start > page.End {
		return fmt.Errorf("page end must be after page start, found start: %v, end: %v", page.Start, page.End)
	}

	audioFile := path.Join(api.SourceDataDir, page.Audio)
	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s (expected location: %s)", page.Audio, audioFile)
	}
	return nil
}

func (api *DBAPI) validatePages(pages []protocol.PagePayload) error {
	seenIDs := make(map[string]bool)
	for i, page := range pages {
		err := api.validatePage(page)
		if err != nil {
			return fmt.Errorf("invalid page id %s : %v", page.ID, err)
		}
		if _, seen := seenIDs[page.ID]; seen {
			return fmt.Errorf("duplicate ids for source data: %s", page.ID)
		}
		if i > 0 {
			prevPage := pages[i-1]
			if prevPage.Start > page.Start {
				return fmt.Errorf("pages must be ordered by start time, found %v before %v", prevPage, page)
			}
			if prevPage.End > page.End {
				return fmt.Errorf("pages must be ordered by end time, found %v before %v", prevPage, page)
			}
			if page.Start < prevPage.End {
				return fmt.Errorf("overlapping pages is not allwed, found %v before %v", prevPage, page)
			}
		}
		seenIDs[page.ID] = true
	}
	return nil
}

func validateAnnotation(anno protocol.AnnotationPayload) error {
	if anno.Page.ID == "" {
		return fmt.Errorf("no id")
	}
	if anno.Page.Audio == "" {
		return fmt.Errorf("no audio")
	}
	if anno.Page.Start > anno.Page.End {
		return fmt.Errorf("annotation end must be after annotation start, found start: %v, end: %v", anno.Page.Start, anno.Page.End)
	}
	for i, chunk := range anno.Chunks {
		if chunk.Start > chunk.End {
			return fmt.Errorf("chunk end must be after chunk start, found start: %v, end: %v", chunk.Start, chunk.End)
		}
		if i > 0 {
			prevChunk := anno.Chunks[i-1]
			if prevChunk.Start > chunk.Start {
				return fmt.Errorf("chunks must be ordered by start time, found %v before %v", prevChunk, chunk)
			}
			if prevChunk.End > chunk.End {
				return fmt.Errorf("chunks must be ordered by end time, found %v before %v", prevChunk, chunk)
			}
			if chunk.Start < prevChunk.End {
				return fmt.Errorf("overlapping chunks is not allowed, found %v before %v", prevChunk, chunk)
			}
		}
		if chunk.CurrentStatus.Name == "" {
			return fmt.Errorf("invalid current status for chunk: %#v", chunk)
		}
		if len(chunk.StatusHistory) > 0 && chunk.CurrentStatus.Name == "" {
			return fmt.Errorf("status history exists, but no current status: %#v", chunk)
		}
	}
	// if len(anno.StatusHistory) > 0 && anno.CurrentStatus.Name == "" {
	// 	return fmt.Errorf("status history exists, but no current status: %#v", anno)
	// }
	return nil
}

func (api *DBAPI) LoadSourceData() ([]protocol.PagePayload, []ValRes, error) {
	files := api.listJSONFiles(api.SourceDataDir)
	var res []protocol.PagePayload
	var vRes []ValRes
	var errRes error
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			var pgs []protocol.PagePayload
			bts, err := os.ReadFile(f)
			if err != nil {
				msg := fmt.Sprintf("couldn't read pages file %s : %v", f, err)
				//return res, fmt.Errorf(msg)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			err = json.Unmarshal(bts, &pgs)
			if err != nil {
				msg := fmt.Sprintf("couldn't unmarshal pages file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			err = api.validatePages(pgs)
			if err != nil {
				msg := fmt.Sprintf("validation error for pages loaded from file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			res = append(res, pgs...)
		}
	}
	return res, vRes, errRes
}

func (api *DBAPI) LoadAnnotationData() (map[string]protocol.AnnotationPayload, []ValRes, error) {
	res := map[string]protocol.AnnotationPayload{}
	var vRes []ValRes
	var errRes error
	files := api.listJSONFiles(api.AnnotationDataDir)
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			bts, err := os.ReadFile(f)
			if err != nil {
				msg := fmt.Sprintf("couldn't read annotation file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			var annotation protocol.AnnotationPayload
			err = json.Unmarshal(bts, &annotation)
			if err != nil {
				msg := fmt.Sprintf("couldn't unmarshal annotation file %s : %v", f, err)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			err = validateAnnotation(annotation)
			if err != nil {
				msg := fmt.Sprintf("invalid json in annotation file %s : %v", f, err)
				//HB errRes = err
				vRes = append(vRes, ValRes{"error", msg})
				continue
			}
			if _, seen := res[annotation.Page.ID]; seen {
				msg := fmt.Sprintf("duplicate page ids for annotation data: %s : %v", f, err)
				errRes = err
				vRes = append(vRes, ValRes{"error", msg})
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

func (api *DBAPI) GetAnnotationData() map[string]protocol.AnnotationPayload {
	return api.annotationData
}

// Pages returns the number of page annotations.
func (api *DBAPI) Pages() int {
	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()
	return len(api.annotationData)
}

func (api *DBAPI) userIDsForLockedPages() []ClientID {
	// save in tmp map to get rid of dupes
	tmpRes := map[ClientID]bool{}
	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
	for _, v := range api.lockMap {
		tmpRes[v] = true
	}
	var res []ClientID
	for k := range tmpRes {
		res = append(res, k)
	}

	return res
}

func (api *DBAPI) Unlock(pageID string, ci ClientID) error {

	if strings.TrimSpace(ci.ID) == "" {
		return fmt.Errorf("dbapi.Unlock: empty ID value: %#v", ci)
	}

	log.Info("[dbapi] Unlock %s %v", pageID, ci)
	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
	lockedBy, locked := api.lockMap[pageID]
	if !locked {
		return fmt.Errorf("%v is not locked", pageID)
	}
	if lockedBy.UserName != ci.UserName {
		return fmt.Errorf("%v is not locked by user %v", pageID, ci)
	}
	delete(api.lockMap, pageID)
	return nil
}

func (api *DBAPI) UnlockUserName(pageID, user string) error {
	log.Info("[dbapi] Unlock %s %s", pageID, user)
	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
	lockedBy, locked := api.lockMap[pageID]
	if !locked {
		return fmt.Errorf("%v is not locked", pageID)
	}
	if lockedBy.UserName != user {
		return fmt.Errorf("%v is not locked by user %s", pageID, user)
	}
	delete(api.lockMap, pageID)
	return nil
}

func (api *DBAPI) UnlockAll(ci ClientID) (int, error) {

	if strings.TrimSpace(ci.ID) == "" {
		return 0, fmt.Errorf("dbapi.UnlockAll: empty ID value: %#v", ci)
	}

	n := 0
	for k, v := range api.lockMap {
		if v == ci {
			err := api.Unlock(k, v)
			if err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func (api *DBAPI) UnlockAllUserName(user string) (int, error) {
	n := 0
	for k, v := range api.lockMap {
		if v.UserName == user {
			err := api.Unlock(k, v)
			if err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func (api *DBAPI) Locked(pageID string) bool {
	api.lockMapMutex.RLock()
	defer api.lockMapMutex.RUnlock()
	_, res := api.lockMap[pageID]
	return res
}

func (api *DBAPI) Lock(pageID string, ci ClientID) error {
	log.Info("[dbapi] Lock %s %v", pageID, ci)
	if strings.TrimSpace(ci.ID) == "" {
		return fmt.Errorf("dbapi.Lock: empty ID value: %#v", ci)
	}

	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
	lockedBy, locked := api.lockMap[pageID]
	if locked {
		return fmt.Errorf("%v is already locked by user %s", pageID, lockedBy)
	}
	api.lockMap[pageID] = ci
	return nil
}

type StatsPerAudio map[string]Stats
type Stats map[string]int

func (api *DBAPI) chunkStats() StatsPerAudio {
	res := StatsPerAudio{}
	allStats := Stats{}
	for _, anno := range api.annotationData {
		audio := strings.TrimSuffix(anno.Page.Audio, filepath.Ext(anno.Page.Audio))
		if _, ok := res[audio]; !ok {
			res[audio] = Stats{}
		}
		audioStats := res[audio]
		for _, chunk := range anno.Chunks {
			audioStats["total"]++
			allStats["total"]++
			status := chunk.CurrentStatus
			if status.Name == "unchecked" {
				audioStats[status.Name]++
				allStats[status.Name]++
			} else {
				audioStats["checked"]++
				allStats["checked"]++
				audioStats["status:"+status.Name]++
				allStats["status:"+status.Name]++
			}
			if len(status.Source) > 0 {
				audioStats["source:"+status.Source]++
				allStats["source:"+status.Source]++
			}
		}
		res[audio] = audioStats
	}
	res["all"] = allStats
	return res
}

type annotationStatus struct {
	name    string
	sources map[string]bool
}

func deriveAnnotationStatus(anno protocol.AnnotationPayload) annotationStatus {
	var seenUnchecked, seenSkip, seenProgress, seenOK bool
	sources := make(map[string]bool)
	for _, chunk := range anno.Chunks {
		if chunk.CurrentStatus.Name == "" {
			continue
		}
		sources[chunk.CurrentStatus.Source] = true
		if chunk.CurrentStatus.Name == "ok" {
			seenOK = true
		} else if chunk.CurrentStatus.Name == "skip" {
			seenSkip = true
		} else if chunk.CurrentStatus.Name == "unchecked" {
			seenUnchecked = true
		} else if chunk.CurrentStatus.Name == "in progress" {
			seenProgress = true
		}
	}
	res := annotationStatus{sources: sources}
	if seenUnchecked {
		res.name = "unchecked"
	} else if seenSkip {
		res.name = "skip"
	} else if seenProgress {
		res.name = "in progress"
	} else if seenOK {
		res.name = "ok"
	} else {
		res.name = "unknown"
	}
	return res
}

type SubProjStats struct {
	//BatchName     string   `json:"batch_name"`
	PagesTot      int            `json:"pages_tot"`
	PagesDone     int            `json:"pages_done"`
	PagesSkip     int            `json:"pages_skip"`
	PagesDelete   int            `json:"pages_delete"`
	PagesLocked   int            `json:"pages_locked"`
	PagesLockedBy []string       `json:"pages_locked_by"`
	DoneByEditor  map[string]int `json:"done_by_editor"`
}

func (api *DBAPI) StatsII() SubProjStats {
	var res SubProjStats
	res.DoneByEditor = map[string]int{}

	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()

	res.PagesTot = api.Pages()

	var pagesDone int
	pageStatus := map[string]int{}
	for _, a := range api.annotationData {
		eds := editors(a)
		derivedStatus := status(a)

		pageStatus[derivedStatus]++

		if !(strings.ToLower(derivedStatus) == "skip" || strings.ToLower(derivedStatus) == "unchecked") {
			pagesDone++
			for _, e := range eds {
				res.DoneByEditor[e]++
			}
			//}

			// if a.CurrentStatus.Name == "normal" || a.CurrentStatus.Name == "" {
			// 	if !hasUncheckedChunk(a) {
			// 		pagesDone++
			// 		for _, e := range eds {
			// 			res.DoneByEditor[e]++
			// 		}
			// 	}
			// } else { // TODO: Everything not "normal" (delete, skip) counts as done
			// 	pagesDone++
			// 	for _, e := range eds {
			// 		res.DoneByEditor[e]++
			// 	}
		}
	}

	res.PagesDone = pagesDone

	for k, v := range pageStatus {
		if k == "delete" {
			res.PagesDelete = v
		}
		if k == "skip" {
			res.PagesSkip = v
		}
	}

	api.lockMapMutex.RLock()
	defer api.lockMapMutex.RUnlock()

	res.PagesLocked = len(api.lockMap)

	lockedBy := map[string]int{}
	for _, v := range api.lockMap {
		lockedBy[v.UserName]++
	}
	for k, v := range lockedBy {
		plb := fmt.Sprintf("%s: %d", k, v)
		res.PagesLockedBy = append(res.PagesLockedBy, plb)
	}

	return res
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

func status(a protocol.AnnotationPayload) string {
	if a.CurrentStatus.Name == "skip" || a.CurrentStatus.Name == "delete" {
		return a.CurrentStatus.Name
	}

	cStatus := map[string]int{}
	for _, c := range a.Chunks {
		s := c.CurrentStatus.Name
		// if s == "" {
		// 	s = "unchecked"
		// }
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

func (api *DBAPI) pageStats() StatsPerAudio {
	res := StatsPerAudio{}
	allStats := Stats{}
	for _, anno := range api.annotationData {
		status := deriveAnnotationStatus(anno)
		audio := strings.TrimSuffix(anno.Page.Audio, filepath.Ext(anno.Page.Audio))
		if _, ok := res[audio]; !ok {
			res[audio] = Stats{}
		}
		audioStats := res[audio]
		if api.Locked(anno.Page.ID) {
			audioStats["locked"]++
			allStats["locked"]++
		}
		for id, ci := range api.lockMap {
			if id == anno.Page.ID {
				audioStats["locked by:"+ci.UserName]++
			}
			allStats["locked by:"+ci.UserName]++
		}

		if status.name == "unchecked" {
			audioStats[status.name]++
			allStats[status.name]++
		} else {
			audioStats["status:"+status.name]++
			allStats["status:"+status.name]++
			audioStats["checked"]++
			allStats["checked"]++
		}
		for source := range status.sources {
			audioStats["source:"+source]++
			allStats["source:"+source]++
		}
		if strings.TrimSpace(anno.Comment) != "" {
			audioStats["comment"]++
			allStats["comment"]++
		}
		res[audio] = audioStats
	}
	res["all"] = allStats
	return res
}

// Stats returns 1) page stats per audio file; 2) chunk stats per audio file; or an error if something is wrong
func (api *DBAPI) Stats() (StatsPerAudio, StatsPerAudio, error) {
	pageStats := api.pageStats()
	chunkStats := api.chunkStats()
	return pageStats, chunkStats, nil
}

const (
	StatusUnchecked  = "unchecked"
	StatusSkip       = "skip"
	StatusOK         = "ok"
	StatusOK2        = "ok2"
	StatusInProgress = "in progress"

	StatusChecked = "checked"

	StatusAny   = "any"
	StatusEmpty = ""

	SourceAny = "any"

	AudioFileAny = "any"
)

func queryMatch(request protocol.QueryRequest, annotation protocol.AnnotationPayload, validator *validation.Validator) (bool, error) {

	//matchingChunks := map[int]bool{}

	var pageStatusMatch = false
	if request.PageStatus == StatusAny || annotation.CurrentStatus.Name == "" || request.PageStatus == annotation.CurrentStatus.Name {
		pageStatusMatch = true
	} // else pageStatusMatch = false

	var statusMatch = false
	if request.Status == StatusAny {
		statusMatch = true
	} else {
		for _, tc := range annotation.Chunks {
			switch request.Status {
			case StatusChecked:
				if tc.CurrentStatus.Name != StatusUnchecked && tc.CurrentStatus.Name != StatusEmpty {
					statusMatch = true
				}
			default:
				if tc.CurrentStatus.Name == request.Status {
					statusMatch = true
				}
			}
		}
	}

	var sourceMatch = false
	if request.Source == SourceAny {
		sourceMatch = true
	} else {
		for _, tc := range annotation.Chunks {
			if tc.CurrentStatus.Source == request.Source {
				sourceMatch = true
			}
		}
	}

	var audioFileMatch = false
	if request.AudioFile == AudioFileAny {
		audioFileMatch = true
	} else {
		audioFileMatch = strings.HasPrefix(annotation.Page.Audio, request.AudioFile)
	}

	var transMatch = false
	if request.TransRE == "" {
		transMatch = true
	} else {
		var compiledRE, err = regexp.Compile(request.TransRE) // TODO inefficient
		if err != nil {
			return false, err
		}
		for _, tc := range annotation.Chunks {
			if compiledRE.MatchString(tc.Trans) {
				transMatch = true
			}
		}
	}

	var validationMatch = false
	if validator == nil || !request.ValidationIssue.HasIssue {
		validationMatch = true
	}

	if request.ValidationIssue.HasIssue && validator != nil {
		valRes := validator.ValidateAnnotation(annotation)
		if len(request.ValidationIssue.RuleNames) == 0 && len(valRes) > 0 {
			validationMatch = true
		} else if len(request.ValidationIssue.RuleNames) > 0 && len(valRes) > 0 {
			rMap := map[string]bool{}
			for _, r := range request.ValidationIssue.RuleNames {
				rMap[r] = true
			}

			for _, vr := range valRes {
				if rMap[vr.RuleName] {
					validationMatch = true
					break
				}
			}
		}
	}

	//fmt.Printf("pageStatusMatch:%t statusMatch:%t sourceMatch:%t audioFileMatch:%t transMatch:%t validationMatch:%t\n", pageStatusMatch, statusMatch, sourceMatch, audioFileMatch, transMatch, validationMatch)

	// res := maps.Keys(matchingChunks)
	// slices.Sort(res)
	// return res, nil

	return pageStatusMatch && statusMatch && sourceMatch && audioFileMatch && transMatch && validationMatch, nil
}

func abs(i int64) int64 {
	if i > 0 {
		return i
	}
	return -i
}

func (api *DBAPI) annotationFromPage(page protocol.PagePayload) protocol.AnnotationPayload {
	annotation, exists := api.annotationData[page.ID]
	if exists {
		return annotation
	}
	return protocol.AnnotationPayload{
		Page: page,
		// CurrentStatus: protocol.Status{Name: "unchecked"},
	}
}

// GetNextPage returns an annotation based on the query request. If an error is found, it returns an empty annotation, and an error. If an error is not found, but there is no page to be found, a message will be returned.
func (api *DBAPI) GetNextPage(query protocol.QueryPayload, currentlyLockedID string, clientID ClientID, lockOnLoad bool) (protocol.AnnotationPayload, string, error) {
	log.Info("[dbapi] GetNextPage query %#v", query)
	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()

	// if debug {
	// 	log.Debug("[dbapi] GetNextPage query: %#v", query)
	// }

	if strings.TrimSpace(clientID.ID) == "" {
		return protocol.AnnotationPayload{}, "", fmt.Errorf("empty ClientID.ID field: %#v", query)
	}
	if strings.TrimSpace(clientID.UserName) == "" {
		return protocol.AnnotationPayload{}, "", fmt.Errorf("empty ClientID.UserName field: %#v", query)
	}

	var currIndex int
	var seenCurrID int64
	if query.RequestIndex != "" {
		var i int
		if query.RequestIndex == "first" {
			i = 0
		} else if query.RequestIndex == "last" {
			i = len(api.sourceData) - 1
		} else {
			reqI, err := strconv.Atoi(query.RequestIndex)
			if err == nil && reqI >= 0 && reqI < len(api.sourceData) {
				i = reqI
			} else {
				return protocol.AnnotationPayload{}, "", fmt.Errorf("invalid request index: %s", query.RequestIndex)
			}
		}
		page := api.sourceData[i]
		if page.ID == currentlyLockedID {
			return protocol.AnnotationPayload{}, "user is already at the requested page", nil
		}
		annotation := api.annotationFromPage(page)
		if lockOnLoad {
			err := api.Lock(annotation.Page.ID, clientID /*ClientID{ID: query.ClientID, UserName: query.UserName}*/)
			if err != nil {
				return protocol.AnnotationPayload{}, "page is already locked", nil
			}
		}
		annotation.Index = int64(i + 1)
		return annotation, "", nil
	} else if query.CurrID != "" {
		seenCurrID = int64(-1)
		for i, page := range api.sourceData {
			if page.ID == query.CurrID {
				currIndex = i
			}
		}
	} else {
		seenCurrID = int64(0)
		currIndex = 0
	}
	for i := currIndex; i >= 0 && i < len(api.sourceData); {
		page := api.sourceData[i]
		if seenCurrID < 0 && page.ID == query.CurrID {
			seenCurrID = 0
			if debug {
				log.Debug("[dbapi] GetNextPage index=%v seenCurrID=%v page.ID=%v stepSize=%v CURR!", i+1, seenCurrID, page.ID, query.StepSize)
			}
		} else {
			annotation := api.annotationFromPage(page)
			if debug {
				log.Debug("[dbapi] GetNextPage index=%v seenCurrID=%v page.ID=%v stepSize=%v derived status=%v", i+1, seenCurrID, page.ID, query.StepSize, deriveAnnotationStatus(annotation).name)
			}
			if seenCurrID >= 0 {
				matches, err := queryMatch(query.Request, annotation, api.validator)

				if err != nil {
					return protocol.AnnotationPayload{}, "", err
				}
				if matches && !api.Locked(page.ID) {
					seenCurrID++
					if query.CurrID == "" || seenCurrID == abs(query.StepSize) {
						if lockOnLoad {
							if api.Locked(annotation.Page.ID) {
								return protocol.AnnotationPayload{}, fmt.Sprintf("%v is already locked", page.ID), nil
							}
							err := api.Lock(annotation.Page.ID, clientID /*ClientID{ID: query.ClientID, UserName: query.UserName}*/)
							if err != nil {
								return protocol.AnnotationPayload{}, "", err
							}

						}
						annotation.Index = int64(i + 1)
						return annotation, "", nil
					}
				}
			}
		}
		if query.StepSize < 0 {
			i--
		} else {
			i++
		}
	}
	var prettyQuery string
	queryJS, err := json.MarshalIndent(query.Request, " ", " ")
	if err == nil {
		prettyQuery = string(queryJS)
	} else {
		prettyQuery = fmt.Sprintf("%#v", query.Request)
	}

	return protocol.AnnotationPayload{}, fmt.Sprintf("no page matching query request\n%s", prettyQuery), nil
}

func (api *DBAPI) ListAudioFiles() []string {
	res := []string{}
	for _, page := range api.sourceData {
		baseName := path.Base(page.Audio)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		if !contains(res, baseName) {
			res = append(res, baseName)
		}
	}
	return res
}

func trimSpace(a *protocol.AnnotationPayload) {
	a.CurrentStatus.Name = strings.TrimSpace(a.CurrentStatus.Name)
	a.CurrentStatus.Source = strings.TrimSpace(a.CurrentStatus.Source)
	for i, c := range a.Chunks {
		a.Chunks[i].CurrentStatus.Name = strings.TrimSpace(c.CurrentStatus.Name)
		a.Chunks[i].CurrentStatus.Source = strings.TrimSpace(c.CurrentStatus.Source)
		a.Chunks[i].Trans = strings.TrimSpace(c.Trans)
		a.Chunks[i].Trans = strings.Replace(a.Chunks[i].Trans, "\u00A0", " ", -1) // No-break space
	}
}

func (api *DBAPI) Save(annotation protocol.AnnotationPayload) error {
	//log.Info("[dbapi] Saved %s\t%s", annotation.Page.ID, annotation.CurrentStatus.Name)

	trimSpace(&annotation)

	api.dbMutex.Lock()
	defer api.dbMutex.Unlock()

	/* SAVE TO CACHE */
	api.annotationData[annotation.Page.ID] = annotation

	/* PRINT TO FILE */

	// create copy for writing, and remove internal index
	saveAnno := annotation
	saveAnno.Index = 0

	f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", annotation.Page.ID))
	writeJSON, err := json.MarshalIndent(saveAnno, " ", " ")
	if err != nil {
		return fmt.Errorf("marhsal failed : %v", err)
	}

	file, err := os.Create(f)
	if err != nil {
		return fmt.Errorf("failed create file %s : %v", f, err)
	}
	defer file.Close()
	file.Write(writeJSON)

	return nil
}

// TODO Move to protocol package?
type Query struct {
	Status  []string
	TransRE *regexp.Regexp
}

func (api *DBAPI) Search(q Query) protocol.QueryResult { //[]protocol.AnnotationPayload {
	res := protocol.QueryResult{} //[]protocol.AnnotationPayload

	// status := map[string]bool{}
	// for _, s := range q.Status {
	// 	status[strings.ToLower(s)] = true
	// }

	api.lockMapMutex.RLock()
	defer api.lockMapMutex.RUnlock()
	for _, a := range api.annotationData {
		matchingChunks := []int{}
		for i, c := range a.Chunks {
			if chunkMatches(q, c) {
				matchingChunks = append(matchingChunks, i)
			}
		}
		if len(matchingChunks) > 0 {
			// TODO set any other useful field of AnnotationPayload
			//page := protocol.AnnotationPayload{Chunks: matchingChunks}

			m := protocol.MatchingPage{
				Page:           a,
				MatchingChunks: matchingChunks,
			}
			res.MatchingPages = append(res.MatchingPages, m)
		}
	}
	log.Info("[dbapi] Search returns %#v", res) // never printed?
	return res
}

// TODO Make more general, so that more parameters can be searched for, in different ways.
func chunkMatches(q Query, c protocol.TransChunk) bool {
	var s, t bool
	s = contains(q.Status, c.CurrentStatus.Name)
	if q.TransRE != nil {
		t = q.TransRE.MatchString(c.Trans)
	}

	if len(q.Status) > 0 && q.TransRE != nil {
		return s && t
	}
	if len(q.Status) > 0 {
		return s
	}

	if q.TransRE != nil {
		return t
	}

	return false
}

func contains(slice []string, s string) bool {
	for _, s0 := range slice {
		if s0 == s {
			return true
		}
	}
	return false
}
