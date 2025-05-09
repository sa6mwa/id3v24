package id3v24

import (
	"encoding/binary"
	"errors"
	"os"
	"strconv"
	"time"

	id3v2 "github.com/bogem/id3v2"
	"github.com/sa6mwa/mp3duration"
)

var (
	ErrBadChapterStartTime error = errors.New("bad chapter start time format (expected HH:MM:SS.mmm)")
	ErrZeroDuration        error = errors.New("duration can not be zero")
)

type Chapter struct {
	Title string `json:"title" yaml:"title,omitempty"`
	Start string `json:"start" yaml:"start,omitempty"` // e.g. "00:05:00.500"
}

func ParseTimeToMillis(t string) (uint32, error) {
	d, err := time.Parse("15:04:05.000", t)
	if err != nil {
		d, err = time.Parse("15:04:05.0", t)
		if err != nil {
			d, err = time.Parse("15:04:05", t)
			if err != nil {
				return 0, ErrBadChapterStartTime
			}
		}
	}
	return uint32((time.Duration(d.Hour())*time.Hour +
		time.Duration(d.Minute())*time.Minute +
		time.Duration(d.Second())*time.Second +
		time.Duration(d.Nanosecond())) / time.Millisecond), nil
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
		m, err := ParseTimeToMillis(ch.Start)
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
