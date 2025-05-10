package id3v24

import (
	"bytes"
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
		t.Errorf("dump and file %s does not compare", testdataFile)
	}

	// if err := os.WriteFile(testdataFile, []byte(dump), 0644); err != nil {
	// 	t.Fatal(err)
	// }
}

func TestGetFFmpegChaptersTXT(t *testing.T) {
	testdataFile := "testdata/chapters.txt"

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

	chaptersTXT, err := GetFFmpegChaptersTXT(duration, chapters)
	if err != nil {
		t.Fatal(err)
	}

	testdata, err := os.ReadFile(testdataFile)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(chaptersTXT, testdata) != 0 {
		t.Errorf("generated chapters.txt and %s does not match", testdataFile)
	}

	// if err := os.WriteFile(testdataFile, chaptersTXT, 0644); err != nil {
	// 	t.Fatal(err)
	// }
}

func TestWriteFFmpegMetadataFile(t *testing.T) {
	testdataFile := "testdata/ffmetadata.txt"
	tm := time.Date(2024, 9, 17, 15, 38, 00, 0, time.Now().Local().Location())
	trackInfo := TrackInfo{
		Title:  "Hello world",
		Album:  "Galaxy",
		Artist: "Universe",
		Genre:  "Podcast",
		Date:   tm,
		Track:  "5",
		Chapters: []Chapter{
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
		},
	}

	ffmetafile, err := WriteFFmpegMetadataFile(30*time.Second, trackInfo)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(ffmetafile)

	ffmetadata, err := os.ReadFile(ffmetafile)
	if err != nil {
		t.Fatal(err)
	}

	testdata, err := os.ReadFile(testdataFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(ffmetadata, testdata) != 0 {
		t.Errorf("generated chapters.txt and %s does not match", testdataFile)
	}

	// a, err := os.Open(ffmetafile)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer a.Close()
	// b, err := os.Create(testdataFile)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer b.Close()
	// if _, err := io.Copy(b, a); err != nil {
	// 	t.Fatal(err)
	// }
}
