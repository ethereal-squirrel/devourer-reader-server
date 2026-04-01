package scanner

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

type AudioTags struct {
	Title           string
	Artist          string
	Album           string
	TrackNumber     int
	DurationSeconds int
	CoverData       []byte
}

func ReadAudioTags(path string) (*AudioTags, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	trackNum, _ := m.Track()

	t := &AudioTags{
		Title:       m.Title(),
		Artist:      m.Artist(),
		Album:       m.Album(),
		TrackNumber: trackNum,
	}

	if pic := m.Picture(); pic != nil {
		t.CoverData = pic.Data
	}

	if strings.ToLower(filepath.Ext(path)) == ".mp3" {
		if d := tlenDuration(m); d > 0 {
			t.DurationSeconds = d
		} else {
			if _, err := f.Seek(0, 0); err == nil {
				t.DurationSeconds = mp3Duration(f)
			}
		}
	}

	return t, nil
}

func tlenDuration(m tag.Metadata) int {
	raw := m.Raw()
	if raw == nil {
		return 0
	}
	v, ok := raw["TLEN"]
	if !ok {
		return 0
	}
	s, ok := v.(string)
	if !ok {
		return 0
	}
	ms, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || ms <= 0 {
		return 0
	}
	return ms / 1000
}

func mp3Duration(f *os.File) int {
	stat, err := f.Stat()
	if err != nil {
		return 0
	}
	fileSize := stat.Size()

	id3Size := readID3v2Size(f)
	if _, err := f.Seek(int64(id3Size), 0); err != nil {
		return 0
	}

	var hdr [4]byte
	if _, err := f.Read(hdr[:]); err != nil {
		return 0
	}
	if !isMP3FrameSync(hdr) {
		return 0
	}

	mpegVer, sampleRate, isMono, samplesPerFrame := parseMP3FrameHeader(hdr)
	if sampleRate == 0 {
		return 0
	}
	bitrateKbps := parseMP3Bitrate(hdr, mpegVer)

	xingOff := xingHeaderOffset(mpegVer, isMono)
	if _, err := f.Seek(int64(id3Size)+4+int64(xingOff), 0); err == nil {
		var marker [4]byte
		if _, err := f.Read(marker[:]); err == nil {
			if s := string(marker[:]); s == "Xing" || s == "Info" {
				var flagBytes [4]byte
				if _, err := f.Read(flagBytes[:]); err == nil {
					if binary.BigEndian.Uint32(flagBytes[:])&0x01 != 0 {
						var fcBytes [4]byte
						if _, err := f.Read(fcBytes[:]); err == nil {
							frameCount := int(binary.BigEndian.Uint32(fcBytes[:]))
							if frameCount > 0 {
								return frameCount * samplesPerFrame / sampleRate
							}
						}
					}
				}
			}
		}
	}

	if bitrateKbps > 0 {
		audioBytes := fileSize - int64(id3Size)
		if audioBytes > 0 {
			return int(audioBytes / (int64(bitrateKbps) * 125))
		}
	}

	return 0
}

func readID3v2Size(f *os.File) int {
	var hdr [10]byte
	n, err := f.Read(hdr[:])
	if err != nil || n < 10 || string(hdr[:3]) != "ID3" {
		f.Seek(0, 0)
		return 0
	}
	size := int(hdr[6])<<21 | int(hdr[7])<<14 | int(hdr[8])<<7 | int(hdr[9])
	return size + 10
}

func isMP3FrameSync(h [4]byte) bool {
	return h[0] == 0xFF && (h[1]&0xE0) == 0xE0
}

func parseMP3FrameHeader(h [4]byte) (mpegVer, sampleRate int, isMono bool, samplesPerFrame int) {
	switch (h[1] >> 3) & 0x03 {
	case 0:
		mpegVer = 25
	case 2:
		mpegVer = 2
	case 3:
		mpegVer = 1
	default:
		return 0, 0, false, 0
	}

	srIndex := (h[2] >> 2) & 0x03
	if srIndex == 3 {
		return 0, 0, false, 0
	}

	var srTable [3]int
	switch mpegVer {
	case 1:
		samplesPerFrame = 1152
		srTable = [3]int{44100, 48000, 32000}
	case 2:
		samplesPerFrame = 576
		srTable = [3]int{22050, 24000, 16000}
	case 25:
		samplesPerFrame = 576
		srTable = [3]int{11025, 12000, 8000}
	}
	sampleRate = srTable[srIndex]

	isMono = (h[3]>>6)&0x03 == 3
	return
}

func parseMP3Bitrate(h [4]byte, mpegVer int) int {
	idx := (h[2] >> 4) & 0x0F
	if idx == 0 || idx == 15 {
		return 0
	}
	var table [16]int
	if mpegVer == 1 {
		table = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	} else {
		table = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	}
	return table[idx]
}

func xingHeaderOffset(mpegVer int, isMono bool) int {
	if mpegVer == 1 {
		if isMono {
			return 17
		}
		return 32
	}
	if isMono {
		return 9
	}
	return 17
}
