package abbrevs

import (
	"fmt"
	"os"
	"testing"
)

func cleanAndCreateTestDir(d string) error {
	err := os.RemoveAll(d)
	if err != nil {
		return fmt.Errorf("could'n clean test dir : %v", err)
	}

	err = os.MkdirAll(d, 0777)
	if err != nil {
		return fmt.Errorf("could'n Mkdir : %v", err)
	}

	return nil
}

func TestLoad(t *testing.T) {

	nd := "nODir665d45454343243244¤%¤¤##"
	am := NewAbbrevManager(nd)

	// nd should not exist
	if am.Load() == nil {
		t.Errorf("Expected error, got nil")
	}

	// create test dir
	baseDir := "/tmp/transtool_testDir/abbrevs001/"

	err := cleanAndCreateTestDir(baseDir)
	if err != nil {
		t.Errorf("failed to create test dir : %v", err)
	}

	am = NewAbbrevManager(baseDir)
	if am.Load() != nil {
		t.Errorf("failed Load() : %v", err)
	}

	// Zero lists
	if w, g := 0, len(am.Lists()); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	l1 := "aNewList"
	if err := am.CreateList(l1); err != nil {
		t.Errorf("expected nil got %v", err)
	}

	// Cannot create same list twice
	if err := am.CreateList(l1); err == nil {
		t.Errorf("expected error got nil")
	}

	// You can delete an empty list file
	if err := am.DeleteListFile(l1); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	// ... put it back
	if err := am.CreateList(l1); err != nil {
		t.Errorf("expected nil got %v", err)
	}

	// One list
	if w, g := 1, len(am.Lists()); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	// add an abbrev
	a := "sthlm"
	e := "Stockholm"
	err = am.Add(l1, a, e)

	if err != nil {
		t.Errorf("%v", err)
	}

	// Cannot add same twice
	err = am.Add(l1, a, e)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// You cannot delete an non-empty list file
	if err := am.DeleteListFile(l1); err == nil {
		t.Errorf("expected error, got nil")
	}

	err = am.Load()
	if err != nil {
		t.Errorf("got error when (re-)loading abbrevs: %v", err)
	}

	lists := am.Lists()
	if w, g := 1, len(lists); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	abbrevsMap1 := am.AbbrevsFor(l1)
	if w, g := 1, len(abbrevsMap1); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := e, abbrevsMap1[a]; w != g {
		t.Errorf("wanted %s got %s", w, g)
	}

	// returns nil on non-existent list
	noSuchMap := am.AbbrevsFor("nø_such_list_(((//&%")
	if noSuchMap != nil {
		t.Errorf("expected nil, got %#v", noSuchMap)
	}

	// Add and delete a new enty
	a2 := "gtb"
	e2 := "Jøttlabårj"
	err = am.Add(l1, a2, e2)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	abbrevsMap2 := am.AbbrevsFor(l1)
	if w, g := 2, len(abbrevsMap2); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := e2, abbrevsMap2[a2]; w != g {
		t.Errorf("wanted %s got %s", w, g)
	}

	err = am.Delete(l1, a2)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	abbrevsMap1 = am.AbbrevsFor(l1)
	if w, g := 1, len(abbrevsMap1); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := e, abbrevsMap1[a]; w != g {
		t.Errorf("wanted %s got %s", w, g)
	}

	l2 := "new_user_"
	err = am.AddCreateIfNotExists(l2, "kr", "kronor")
	if err != nil {
		t.Errorf("%v", err)
	}
	abbrevsMap3 := am.AbbrevsFor(l2)
	if w, g := 1, len(abbrevsMap3); w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

	err = am.Load()
	if err != nil {
		t.Errorf("got error when (re-)loading abbrevs: %v", err)
	}

}
