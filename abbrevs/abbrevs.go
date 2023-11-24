package abbrevs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stts-se/transtool-open/log"
)

// NL 20210809 The code below is lifted from https://github.com/stts-se/tord3000
// A lot of code has been commented out, since it's currently not used

var verb = true

var ext = ".abb"

var deletePrefix = "DELETE:>>"

// AbbrevManager keeps track of the abbreviations database. It *must*
// exist only one instance of this struct in order for it to be
// thread safe.
type AbbrevManager struct {
	sync.Mutex
	baseDir string
	lists   map[string]map[string]string
}

func NewAbbrevManager(baseDir string) AbbrevManager {
	return AbbrevManager{
		baseDir: baseDir,
		lists:   make(map[string]map[string]string),
	}
}

// TODO warn for duplicate abbrevs with different expansions

// Load reads the abbreviation list files of the 'baseDir' supplied
// when creating a AbbrevManager.
func (am *AbbrevManager) Load() error {

	am.Lock()
	defer am.Unlock()

	d, err := os.Stat(am.baseDir)
	if err != nil {
		return fmt.Errorf("loading error: %v", err)
	}
	if !d.Mode().IsDir() {
		return fmt.Errorf("not a directory: '%s'", am.baseDir)
	}

	files, err := filepath.Glob(filepath.Join(am.baseDir, "/*"+ext))
	if err != nil {
		return fmt.Errorf("failed listing abb files: %v", err)
	}

	for _, f := range files {

		if verb {
			log.Info("[abbrevs] loading file %s", f)
		}
		bytes, err := ioutil.ReadFile(filepath.Clean(f))
		if err != nil {
			return fmt.Errorf("failed loading file : %v", err)
		}

		fn := filepath.Base(f)
		listName := strings.TrimSuffix(fn, ext)
		// if listName does not already exist, initialise it
		if _, ok := am.lists[listName]; !ok {
			am.lists[listName] = make(map[string]string)
		}

		lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
		for _, l := range lines {
			l := strings.TrimSpace(l)

			// Skip empty lines
			if l == "" {
				continue
			}

			if strings.HasPrefix(l, deletePrefix) {
				l0 := strings.TrimPrefix(l, deletePrefix)
				fs := strings.Split(l0, "\t")

				m, ok := am.lists[listName]
				if ok {
					if len(fs) < 1 {
						// TODO log and continue
						return fmt.Errorf("no abbrev for DELETE line")
					}

					delete(m, fs[0])
				} else {
					// TODO log and continue
					return fmt.Errorf("could not delete non-existing abbrev '%s'", fs[0])
				}
			} else {
				fs := strings.Split(l, "\t")
				if len(fs) != 2 {
					// TODO skip and log?
					// TODO allow comments?
					return fmt.Errorf("faulty number of fields in line: %s in %s", l, listName)
				}

				// TODO skip and log?
				if len(strings.TrimSpace(fs[0])) == 0 {
					return fmt.Errorf("cannot have empty field: '%s'", l)
				}
				if len(strings.TrimSpace(fs[1])) == 0 {
					return fmt.Errorf("cannot have empty field: '%s'", l)
				}

				if _, ok := am.lists[listName][fs[0]]; ok {
					log.Info("[abbrevs] Skipping duplicate abbrev\t%s\t%s\n", fs[0], fs[1])
					continue
				}

				am.lists[listName][fs[0]] = fs[1]
			}
		}

	}

	return nil
}

// ListLength holds the list name and number of abbreviations
type ListLength struct {
	Name   string `json:"name"`
	Length int    `json:"length"`
}

func (am *AbbrevManager) ListsWithLength() []ListLength {
	var res []ListLength

	for l, a := range am.lists {
		res = append(res, ListLength{Name: l, Length: len(a)})
	}

	return res
}

func (am *AbbrevManager) Lists() []string {
	res := []string{}
	for l := range am.lists {
		res = append(res, l)
	}

	return res
}

func (am *AbbrevManager) CreateList(l string) error {

	am.Lock()
	defer am.Unlock()

	if _, ok := am.lists[l]; ok {
		return fmt.Errorf("list '%s' already exists", l)
	}

	path := filepath.Join(am.baseDir, l+ext)

	//fmt.Printf("PATH: %v\n", path)

	if _, err := os.Stat(path); !os.IsNotExist(err) { //err == nil {
		return fmt.Errorf("the list file '%v' already exists", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create list '%s' : %v", l, err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to close file '%s' : %v", l, err)
	}

	am.lists[l] = make(map[string]string)

	return nil
}

// TODO Maybe should return (bool, error)?
func (am *AbbrevManager) DeleteListFile(l string) error {
	l = strings.TrimSpace(l)
	am.Lock()
	defer am.Unlock()

	path := filepath.Join(am.baseDir, l+ext)

	// TODO Check also the lenght of the actual file?
	// TODO: Cannot easily check if file is "empty" since file can contain "DELETE" lines

	// empty, err := fileEmpty(path)
	// if err != nil {
	// 	msg := fmt.Sprintf("failed to check if file is empty : %v", err)
	// 	return fmt.Errorf(msg)
	// }

	//fmt.Println("EMPTY", path)
	//fmt.Println("EMPTY", empty)

	//if len(am.lists[l]) > 0 || !empty {
	if len(am.lists[l]) > 0 {
		msg := fmt.Sprintf("cannot delete non-empty abbreviations list '%s'", l)
		//log.Error(msg)
		return fmt.Errorf(msg)
	}

	delete(am.lists, l)
	return os.Remove(path)
}

func (am *AbbrevManager) AbbrevsFor(list string) map[string]string {
	am.Lock()
	defer am.Unlock()
	return am.lists[list]
}

func (am *AbbrevManager) Add(listName, abbrev, expansion string) error {
	return am.addOrDelete(listName, abbrev, expansion, "")
}

// AddCreateIfNotExists creates list listName if it does not already exist
func (am *AbbrevManager) AddCreateIfNotExists(listName, abbrev, expansion string) error {

	var exists bool
	for _, list := range am.Lists() {
		// TODO: All listNames should be lower case?
		//if strings.ToLower(listName) == strings.ToLower(list) {
		if listName == list {
			exists = true
			break
		}
	}

	if !exists {
		err := am.CreateList(listName)
		if err != nil {
			return fmt.Errorf("abbrevs.AbbrevManager.AddCreateIfNotExists failed : %v", err)
		}
	}

	return am.addOrDelete(listName, abbrev, expansion, "")
}

func (am *AbbrevManager) Delete(listName, abbrev string) error {
	return am.addOrDelete(listName, abbrev, "", deletePrefix)
}

// TODO Maybe a little odd to have an add or delete function (but both write to the same file
func (am *AbbrevManager) addOrDelete(listName, abbrev, expansion, linePrefix string) error {
	abbrev = strings.TrimSpace(abbrev)
	expansion = strings.TrimSpace(expansion)
	//linePrefix = strings.TrimSpace(linePrefix)

	am.Lock()
	defer am.Unlock()

	l, ok := am.lists[listName]
	if !ok {
		return fmt.Errorf("list '%s' doesn't exist", listName)
	}

	// TODO if more line prefixes are added, these need to be checked for.
	// Alternatively, check that linePrefix is not the empty string
	if _, ok := l[abbrev]; ok && linePrefix != deletePrefix {
		return fmt.Errorf("abbreviation '%s' already exists in list '%s'", abbrev, listName)
	}

	path := filepath.Join(am.baseDir, listName+ext)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file : %v", err)
	}
	defer f.Close()

	abbr := fmt.Sprintf("%s\t%s\n", abbrev, expansion)
	if strings.TrimSpace(linePrefix) != "" {
		abbr = linePrefix + abbr
	}

	_, err = f.WriteString(abbr)
	if err != nil {
		return fmt.Errorf("failed to write abbreviation '%s' to list '%s' : %v", abbrev, listName, err)
	}

	// Delete or add to map, depending on value of linePrefix
	if linePrefix == deletePrefix {
		delete(am.lists[listName], abbrev)
	} else {
		am.lists[listName][abbrev] = expansion
	}
	return nil
}
