package id3v24

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	id3v2 "github.com/bogem/id3v2"
	"github.com/sa6mwa/mp3duration"
)

var (
	ErrBadChapterStartTime error = errors.New("bad chapter start time format (expected HH:MM:SS.mmm)")
	ErrZeroDuration        error = errors.New("duration can not be zero")
)

type TrackInfo struct {
	Title       string    `json:"title" yaml:"title,omitempty"`
	Album       string    `json:"album" yaml:"album,omitempty"`
	Artist      string    `json:"artist" yaml:"artist,omitempty"`
	Genre       string    `json:"genre" yaml:"genre,omitempty"`
	Year        string    `json:"year" yaml:"year,omitempty"`
	Date        time.Time `json:"date" yaml:"date,omitempty"` // yyyy-mm-dd
	Track       string    `json:"track" yaml:"track,omitempty"`
	Comment     string    `json:"comment" yaml:"comment,omitempty"`
	Description string    `json:"description" yaml:"description,omitempty"`
	Language    string    `json:"language" yaml:"language,omitempty"`
	Copyright   string    `json:"copyright" yaml:"copyright,omitempty"`
	CoverJPEG   string    `json:"coverJPEG" yaml:"coverJPEG,omitempty"`
	Chapters    []Chapter `json:"chapters" yaml:"chapters,omitempty"`
}

type Chapter struct {
	Title string `json:"title" yaml:"title,omitempty"`
	Start string `json:"start" yaml:"start,omitempty"` // e.g. "00:05:00.500"
}

func StringTimeToMillis(t string) (uint32, error) {
	d, err := StringTimeToTime(t)
	if err != nil {
		return 0, err
	}
	return uint32((time.Duration(d.Hour())*time.Hour +
		time.Duration(d.Minute())*time.Minute +
		time.Duration(d.Second())*time.Second +
		time.Duration(d.Nanosecond())) / time.Millisecond), nil
}

func StringTimeToTime(t string) (time.Time, error) {
	d, err := time.Parse("15:04:05.000", t)
	if err != nil {
		d, err = time.Parse("15:04:05.0", t)
		if err != nil {
			d, err = time.Parse("15:04:05", t)
			if err != nil {
				return time.Time{}, ErrBadChapterStartTime
			}
		}
	}
	return d, nil
}

func GetMP3Duration(mp3path string) (time.Duration, error) {
	f, err := os.Open(mp3path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	d, err := mp3duration.Read(f)
	if err != nil {
		return 0, err
	}
	return d.TimeDuration, nil
}

// TextFrame returns an UTF-16 ID3v2.4 Text Frame from title string.
func TextFrame(title string) []byte {
	frame := []byte{0x01}             // UTF-16 with BOM (0x01)
	frame = append(frame, 0xFF, 0xFE) // BOM (byte order mark)
	for _, r := range title {
		frame = append(frame, byte(r), 0x00) // UTF-16LE encoding
	}
	return frame
}

// AddCHAPAndCTOC adds each CHAP and a final CTOC frame to tag from a
// slice of Chapter structs. duration is an Info struct returned by
// mp3duration.Read or ReadFile as AddCHAPAndCTOC need to know the
// duration of the underlying MP3 in order to calculate end of last
// chapter. If chapters is an empty slice, no frames will be
// added. Returns error if something failed, in which case tag is to
// be considered corrupt (should not be saved via tag.Save).
func AddCHAPAndCTOC(duration mp3duration.Info, tag *id3v2.Tag, chapters []Chapter) error {
	if len(chapters) == 0 {
		return nil
	}
	if duration.TimeDuration == 0 {
		return ErrZeroDuration
	}
	millis := uint32(duration.TimeDuration / time.Millisecond)

	starts := make([]uint32, len(chapters))
	chapterIDs := []string{}

	for i, ch := range chapters {
		m, err := StringTimeToMillis(ch.Start)
		if err != nil {
			return err
		}
		starts[i] = m
	}

	// CHAP encoding loop
	for i, ch := range chapters {
		start := starts[i]
		var end uint32
		if i < len(chapters)-1 {
			end = starts[i+1]
		} else {
			end = millis
		}
		chapterID := strconv.Itoa(i + 1)
		body := []byte{}
		body = append(body, []byte(chapterID)...)
		body = append(body, 0x00)
		timeBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(timeBuf, start)
		body = append(body, timeBuf...)
		binary.BigEndian.PutUint32(timeBuf, end)
		body = append(body, timeBuf...)
		body = append(body, []byte{0xFF, 0xFF, 0xFF, 0xFF}...) // start offset
		body = append(body, []byte{0xFF, 0xFF, 0xFF, 0xFF}...) // end offset

		titleFrame := TextFrame(ch.Title)
		titleHeader := []byte("TIT2")
		lengthBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lengthBuf, uint32(len(titleFrame)))
		titleHeader = append(titleHeader, lengthBuf...)
		titleHeader = append(titleHeader, []byte{0x00, 0x00}...)
		body = append(body, titleHeader...)
		body = append(body, titleFrame...)

		tag.AddFrame("CHAP", id3v2.UnknownFrame{Body: body})
		chapterIDs = append(chapterIDs, chapterID)
	}

	// Add CTOC frame
	ctocBody := []byte("toc\x00")
	ctocBody = append(ctocBody, []byte{0x01, 0x00}...)
	ctocBody = append(ctocBody, byte(len(chapterIDs)))
	for _, id := range chapterIDs {
		ctocBody = append(ctocBody, []byte(id)...)
		ctocBody = append(ctocBody, 0x00)
	}
	tag.AddFrame("CTOC", id3v2.UnknownFrame{Body: ctocBody})
	return nil
}

// AddCoverJPEG adds a cover picture (jpegPath) to tag or return
// error.
func AddCoverJPEG(tag *id3v2.Tag, jpegPath string) error {
	imgData, err := os.ReadFile(jpegPath)
	if err != nil {
		return err
	}
	picFrame := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingISO,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Cover",
		Picture:     imgData,
	}
	tag.AddAttachedPicture(picFrame)
	return nil
}

// WriteID3v2Tag writes everything this package is designed for;
// title, album, arist, genre, year, cover picture (jpeg), and
// chapters. If any field is empty (zero length or empty slice, etc),
// it will not be added to the tag. The output mp3 will be modified.
func WriteID3v2Tag(mp3file string, input TrackInfo) error {
	di, err := mp3duration.ReadFile(mp3file)
	if err != nil {
		return err
	}
	tag, err := id3v2.Open(mp3file, id3v2.Options{Parse: false})
	if err != nil {
		return err
	}
	defer tag.Close()
	// Important
	tag.SetVersion(4)
	// Set frames unless empty...
	if len([]rune(input.Title)) > 0 {
		tag.SetTitle(input.Title)
	}
	if len([]rune(input.Album)) > 0 {
		tag.SetAlbum(input.Album)
	}
	if len([]rune(input.Artist)) > 0 {
		tag.SetArtist(input.Artist)
	}
	if len([]rune(input.Genre)) > 0 {
		tag.SetGenre(input.Genre)
	}
	if len([]rune(input.Year)) > 0 {
		tag.SetYear(input.Year)
	}
	if len([]rune(input.CoverJPEG)) > 0 {
		if err := AddCoverJPEG(tag, input.CoverJPEG); err != nil {
			return err
		}
	}
	if len(input.Chapters) > 0 {
		if err := AddCHAPAndCTOC(di, tag, input.Chapters); err != nil {
			return err
		}
	}
	// Save tag information
	if err := tag.Save(); err != nil {
		return err
	}
	return nil
}

// GetFFmpegChaptersTXT returns a chapters.txt file for use with
// FFmpeg when generating e.g m4b files. Maybe strange to also support
// ffmpeg and m4b in a package for MP3 ID3 tags, but the functionality
// is already here and chapters in m4b is much better. Returns a
// chapters.txt as a byte slice or error if something failed.
func GetFFmpegChaptersTXT(duration mp3duration.Info, chapters []Chapter) ([]byte, error) {
	var output []byte = []byte(";FFMETADATA1\n")
	if len(chapters) == 0 {
		return nil, nil
	}
	if duration.TimeDuration == 0 {
		return nil, ErrZeroDuration
	}
	millis := uint32(duration.TimeDuration / time.Millisecond)
	starts := make([]uint32, len(chapters))
	for i, ch := range chapters {
		m, err := StringTimeToMillis(ch.Start)
		if err != nil {
			return nil, err
		}
		starts[i] = m
	}
	for i, ch := range chapters {
		start := starts[i]
		var end uint32
		if i < len(chapters)-1 {
			end = starts[i+1]
		} else {
			end = millis
		}
		output = append(output, []byte(fmt.Sprintf("\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=%d\nEND=%d\ntitle=%s\n",
			start, end, ch.Title,
		))...)
	}
	return output, nil
}

// WriteFFmpegChaptersTXT returns a temporary (os.CreateTemp)
// ffmpeg-compatible chapters.txt file for use if generating e.g an
// m4b instead of an mp3. Returns full path to tempfile or error if
// something failed.
func WriteFFmpegChaptersTXT(duration mp3duration.Info, chapters []Chapter) (string, error) {
	var removeTempfile bool
	chaptersTXT, err := GetFFmpegChaptersTXT(duration, chapters)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "*-chapters.txt")
	if err != nil {
		return "", err
	}
	defer func() {
		f.Close()
		if removeTempfile {
			os.Remove(f.Name())
		}
	}()
	if _, err := f.Write(chaptersTXT); err != nil {
		removeTempfile = true
		return "", err
	}
	return f.Name(), nil
}

// WriteFFmpegMetadataFile returns a temporary (os.CreateTemp)
// ffmpeg-compatible metadata file for use with illustrative example:
//
//	ffmpeg -i input.flac output.m4a
//	ffmpeg -i output.m4a -i metadata.txt -map_metadata 1 -codec copy final_output.m4a
//
// Returns full path to tempfile or error if something failed.
func WriteFFmpegMetadataFile(duration time.Duration, input TrackInfo) (string, error) {
	var removeTempfile bool
	var output []byte = []byte(";FFMETADATA1\n")
	chaptersTXT, err := GetFFmpegChaptersTXT(mp3duration.Info{TimeDuration: duration}, input.Chapters)
	if err != nil {
		return "", err
	}
	if chaptersTXT == nil {
		chaptersTXT = make([]byte, 0)
	} else {
		// Remove ";FFMETADATA" line from chaptersTXT
		chaptersTXT = bytes.Replace(chaptersTXT, output, nil, 1)
	}
	f, err := os.CreateTemp("", "*-ffmetadata.txt")
	if err != nil {
		return "", err
	}
	defer func() {
		f.Close()
		if removeTempfile {
			os.Remove(f.Name())
		}
	}()
	kvpairs := []map[string]string{
		{"title": input.Title},
		{"album": input.Album},
		{"artist": input.Artist},
		{"genre": input.Genre},
		{"track": input.Track},
		{"comment": input.Comment},
		{"language": input.Language},
		{"description": input.Description},
		{"copyright": fmt.Sprintf("Copyright %s %s", input.Date.Format("2006"), input.Artist)},
	}
	if !input.Date.IsZero() {
		kvpairs = append(kvpairs, map[string]string{"date": input.Date.Format("2006-01-02")})
	}
	for i := range kvpairs {
		for k, v := range kvpairs[i] {
			if len([]rune(v)) > 0 {
				appendKVPair(&output, k, v)
			}
		}
	}
	// Append chapters
	output = append(output, chaptersTXT...)
	if _, err := f.Write(output); err != nil {
		removeTempfile = true
		return "", err
	}
	return f.Name(), nil
}

func appendKVPair(output *[]byte, key, value string) {
	clean := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return -1 // remove linefeeds
		}
		return r
	}, value)
	*output = append(*output, []byte(key+"="+strings.TrimSpace(clean)+"\n")...)
}
