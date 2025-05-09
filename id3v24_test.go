package id3v24

import (
	"os"
	"strings"
	"testing"
	"time"

	id3v2 "github.com/bogem/id3v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/sa6mwa/mp3duration"
)

func TestAddCHAPAndCTOC(t *testing.T) {
	testdataFile := "testdata/addchapandctoc"

	tag := id3v2.NewEmptyTag()

	tag.SetArtist("John Doe")
	tag.SetTitle("Test Title")
	tag.SetAlbum("Hello World")

	chapters := []Chapter{
		Chapter{
			Title: "Chapter 1",
			Start: "00:00:00.000",
		},
		Chapter{
			Title: "Chapter 2",
			Start: "00:00:10",
		},
		Chapter{
			Title: "Chapter 3",
			Start: "00:00:20.5",
		},
	}

	duration := mp3duration.Info{
		TimeDuration: 30 * time.Second,
	}

	if err := AddCHAPAndCTOC(duration, tag, chapters); err != nil {
		t.Fatal(err)
	}

	cfg := spew.ConfigState{
		Indent:                  " ",
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		SortKeys:                true,
		SpewKeys:                false,
	}

	dump := cfg.Sdump(tag)

	testdata, err := os.ReadFile(testdataFile)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(dump, string(testdata)) != 0 {
		t.Error("dump and testdata does not compare")
	}

	// if err := os.WriteFile(testdataFile, []byte(dump), 0644); err != nil {
	// 	t.Fatal(err)
	// }
}
