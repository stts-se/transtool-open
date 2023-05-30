package dbapi

import (
	//"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stts-se/transtool/protocol"
)

func TestSearch(t *testing.T) {
	// TODO create proper setup with corresponding file directories
	var db = DBAPI{
		dbMutex:      &sync.RWMutex{},
		lockMapMutex: &sync.RWMutex{},

		annotationData: map[string]protocol.AnnotationPayload{

			"s1": {
				CurrentStatus: protocol.Status{Name: "delete"},
				Chunks: []protocol.TransChunk{
					{Trans: "trans1", CurrentStatus: protocol.Status{Name: "ok"}},
					{Trans: "trans2", CurrentStatus: protocol.Status{Name: "skip"}},
					{Trans: "trans3", CurrentStatus: protocol.Status{Name: "unchecked"}},
				},
			},
			"s2": {
				CurrentStatus: protocol.Status{Name: "normal"},
				Chunks: []protocol.TransChunk{
					{Trans: "trans4"},
					{Trans: "trans5"},
					{Trans: "trans6"},
				},
			},
		},
	}

	res1 := db.Search(Query{})
	if w, g := 0, len(res1.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res2 := db.Search(Query{Status: []string{"ok"}})
	if w, g := 1, len(res2.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	//res2 := db.Search(Query{Status: []string{"ok"}})
	if w, g := 1, len(res2.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res3 := db.Search(Query{Status: []string{"ok", "skip"}})
	if w, g := 1, len(res3.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := 2, len(res3.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res4 := db.Search(Query{Status: []string{"ok", "skip", "unchecked"}})
	if w, g := 3, len(res4.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res5 := db.Search(Query{TransRE: regexp.MustCompile("XYXXZZYZZWWQQ")})
	if w, g := 0, len(res5.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res6 := db.Search(Query{TransRE: regexp.MustCompile("4")})
	if w, g := 1, len(res6.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	// Index 0 (first chunk)
	if w, g := 0, res6.MatchingPages[0].MatchingChunks[0]; g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	// two pages match
	res7 := db.Search(Query{TransRE: regexp.MustCompile("[14]")})

	//fmt.Printf("RES: %#v\n\n", res7)

	//inx := res7.MatchingPages[0].MatchingChunks
	//fmt.Printf("MatchIndex: %#v\n\n", inx)

	if w, g := 2, len(res7.MatchingPages); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := 1, len(res7.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	// Check index for chunk
	if w, g := 1, len(res7.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	if w, g := 1, len(res7.MatchingPages[1].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	res8 := db.Search(Query{TransRE: regexp.MustCompile("[2]")})
	// Check index for chunk
	if w, g := 1, len(res8.MatchingPages[0].MatchingChunks); g != w {
		t.Errorf("wanted %d got %d", w, g)
	}

	// res9 := db.Search(Query{PageStatus: "any"})
	// // Check index for chunk
	// if w, g := 2, len(res9); g != w {
	// 	t.Errorf("wanted %d got %d", w, g)
	// }

}

func TestPageStatus(t *testing.T) {

	var p1 = protocol.PagePayload{
		ID: "p1",
		Chunk: protocol.Chunk{
			Start: 0,
			End:   49,
		},
	}

	var p2 = protocol.PagePayload{
		ID: "p2",
		Chunk: protocol.Chunk{
			Start: 50,
			End:   100,
		},
	}

	var db = &DBAPI{
		dbMutex:      &sync.RWMutex{},
		lockMapMutex: &sync.RWMutex{},

		sourceData: []protocol.PagePayload{
			p1,
			p2,
		},
		annotationData: map[string]protocol.AnnotationPayload{

			"p1": {
				Page:          p1,
				CurrentStatus: protocol.Status{Name: "delete", Source: "s1"},
				Chunks: []protocol.TransChunk{
					{Chunk: protocol.Chunk{Start: 0, End: 19},
						Trans: "trans1", CurrentStatus: protocol.Status{Name: "ok"}},
					{Chunk: protocol.Chunk{Start: 20, End: 29},
						Trans: "trans2", CurrentStatus: protocol.Status{Name: "skip"}},
					{Chunk: protocol.Chunk{Start: 30, End: 49},
						Trans: "trans3", CurrentStatus: protocol.Status{Name: "unchecked"}},
				},
			},
			"p2": {
				Page:          p2,
				CurrentStatus: protocol.Status{Name: "normal", Source: "s1"},
				Chunks: []protocol.TransChunk{
					{Chunk: protocol.Chunk{Start: 50, End: 59},
						Trans: "trans4"},
					{Chunk: protocol.Chunk{Start: 60, End: 69},
						Trans: "trans5"},
					{Chunk: protocol.Chunk{Start: 70, End: 79},
						Trans: "trans6"},
				},
			},
		},
	}

	q := protocol.QueryPayload{
		//RequestIndex: "",
		//CurrID:       "",
		Request: protocol.QueryRequest{
			PageStatus: "normal",
			Status:     "any",
			Source:     "any",
		},
	}
	nextAnno, _, err := db.GetNextPage(q, "", ClientID{ID: "ididid", UserName: "s1"}, false)
	if err != nil {
		t.Errorf("%v", err)
	}

	if w, g := "p2", nextAnno.Page.ID; w != g {
		t.Errorf("wanted %s got %s", w, g)
	}

	stats := db.StatsII()
	if w, g := 1, stats.PagesDelete; w != g {
		t.Errorf("wanted %d got %d", w, g)
	}
	if w, g := 0, stats.PagesSkip; w != g {
		t.Errorf("wanted %d got %d", w, g)
	}

}

func TestTrimSpace(t *testing.T) {

	a := protocol.AnnotationPayload{
		CurrentStatus: protocol.Status{
			Name:   " funky status ",
			Source: " flashy source   ",
		},
		Chunks: []protocol.TransChunk{
			{Chunk: protocol.Chunk{Start: 50, End: 59},
				Trans: "  trans  "},
			{Chunk: protocol.Chunk{Start: 59, End: 69},
				Trans: "  trans\u00A0trans\u00A0  "},
		},
	}

	if !strings.Contains(a.Chunks[1].Trans, "\u00A0") {
		t.Errorf("Expected no-break space")
	}

	trimSpace(&a)

	if strings.Contains(a.Chunks[1].Trans, "\u00A0") {
		t.Errorf("Did not expect no-break space")
	}

	if a.CurrentStatus.Name != "funky status" {
		t.Errorf("Expected trimmed string, got '%s'", a.CurrentStatus.Name)
	}
	if a.CurrentStatus.Source != "flashy source" {
		t.Errorf("Expected trimmed string, got '%s'", a.CurrentStatus.Source)
	}

	if a.Chunks[0].Trans != "trans" {
		t.Errorf("Expected trimmed string, got '%s'", a.Chunks[0].Trans)
	}

	if a.Chunks[1].Trans != "trans trans" {
		t.Errorf("Expected trimmed string, got '%s'", a.Chunks[1].Trans)
	}

}

//func dummy() { fmt.Println() }
