package index

import (
	"bytes"
	"io"
	"math"
	"os"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type EncodingResult struct {
	Encoding   string  `json:"encoding"`
	Confidence float64 `json:"confidence"`
	HasBOM     bool    `json:"has_bom"`
}

type encodingCandidate struct {
	name       string
	encoding   encoding.Encoding
	confidence float64
}

const (
	minSampleSize = 512
	maxSampleSize = 8192
)

func DetectEncoding(data []byte) EncodingResult {
	if len(data) == 0 {
		return EncodingResult{Encoding: "utf-8", Confidence: 1.0}
	}

	result := detectBOM(data)
	if result.Confidence == 1.0 {
		return result
	}

	result = detectByStatisticalAnalysis(data)
	return result
}

func detectBOM(data []byte) EncodingResult {
	if len(data) >= 3 {
		if bytes.Equal(data[:3], []byte{0xEF, 0xBB, 0xBF}) {
			return EncodingResult{Encoding: "utf-8", Confidence: 1.0, HasBOM: true}
		}
	}

	if len(data) >= 2 {
		if bytes.Equal(data[:2], []byte{0xFF, 0xFE}) {
			return EncodingResult{Encoding: "utf-16le", Confidence: 1.0, HasBOM: true}
		}
		if bytes.Equal(data[:2], []byte{0xFE, 0xFF}) {
			return EncodingResult{Encoding: "utf-16be", Confidence: 1.0, HasBOM: true}
		}
	}

	return EncodingResult{Encoding: "", Confidence: 0}
}

func detectByStatisticalAnalysis(data []byte) EncodingResult {
	sample := data
	if len(sample) > maxSampleSize {
		sample = data[:maxSampleSize]
	}

	if len(sample) < minSampleSize && isASCII(sample) {
		return EncodingResult{Encoding: "ascii", Confidence: 1.0}
	}

	if isValidUTF8Sequence(sample) {
		return EncodingResult{Encoding: "utf-8", Confidence: 0.95}
	}

	candidates := []encodingCandidate{
		{name: "ascii", encoding: nil, confidence: scoreASCII(sample)},
		{name: "utf-8", encoding: nil, confidence: scoreUTF8(sample)},
		{name: "windows-1252", encoding: charmap.Windows1252, confidence: scoreWindows1252(sample)},
		{name: "iso-8859-1", encoding: charmap.ISO8859_1, confidence: scoreISO88591(sample)},
		{name: "iso-8859-2", encoding: charmap.ISO8859_2, confidence: scoreISO88592(sample)},
		{name: "iso-8859-5", encoding: charmap.ISO8859_5, confidence: scoreISO88595(sample)},
		{name: "iso-8859-6", encoding: charmap.ISO8859_6, confidence: scoreISO88596(sample)},
		{name: "iso-8859-7", encoding: charmap.ISO8859_7, confidence: scoreISO88597(sample)},
		{name: "iso-8859-8", encoding: charmap.ISO8859_8, confidence: scoreISO88598(sample)},
		{name: "windows-1250", encoding: charmap.Windows1250, confidence: scoreWindows1250(sample)},
		{name: "windows-1251", encoding: charmap.Windows1251, confidence: scoreWindows1251(sample)},
		{name: "windows-1253", encoding: charmap.Windows1253, confidence: scoreWindows1253(sample)},
		{name: "windows-1254", encoding: charmap.Windows1254, confidence: scoreWindows1254(sample)},
		{name: "windows-1255", encoding: charmap.Windows1255, confidence: scoreWindows1255(sample)},
		{name: "windows-1256", encoding: charmap.Windows1256, confidence: scoreWindows1256(sample)},
		{name: "windows-1257", encoding: charmap.Windows1257, confidence: scoreWindows1257(sample)},
		{name: "windows-1258", encoding: charmap.Windows1258, confidence: scoreWindows1258(sample)},
		{name: "koi8r", encoding: charmap.KOI8R, confidence: scoreKOI8R(sample)},
		{name: "koi8u", encoding: charmap.KOI8U, confidence: scoreKOI8U(sample)},
		{name: "utf-16le", encoding: unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), confidence: scoreUTF16LE(sample)},
		{name: "utf-16be", encoding: unicode.UTF16(unicode.BigEndian, unicode.UseBOM), confidence: scoreUTF16BE(sample)},
		{name: "shift-jis", encoding: japanese.ShiftJIS, confidence: scoreShiftJIS(sample)},
		{name: "euc-jp", encoding: japanese.EUCJP, confidence: scoreEUCJP(sample)},
		{name: "iso-2022-jp", encoding: japanese.ISO2022JP, confidence: scoreISO2022JP(sample)},
		{name: "gbk", encoding: simplifiedchinese.GBK, confidence: scoreGBK(sample)},
		{name: "gb18030", encoding: simplifiedchinese.GB18030, confidence: scoreGB18030(sample)},
		{name: "gb2312", encoding: simplifiedchinese.HZGB2312, confidence: scoreGB2312(sample)},
		{name: "big5", encoding: traditionalchinese.Big5, confidence: scoreBig5(sample)},
		{name: "euc-kr", encoding: korean.EUCKR, confidence: scoreEUCKR(sample)},
	}

	best := EncodingResult{Encoding: "utf-8", Confidence: 0.3}

	for _, cand := range candidates {
		if cand.confidence > best.Confidence {
			best.Encoding = cand.name
			best.Confidence = cand.confidence
		}
	}

	return best
}

func isASCII(data []byte) bool {
	for _, b := range data {
		if b > 127 {
			return false
		}
	}
	return true
}

func isValidUTF8Sequence(data []byte) bool {
	hasNonASCII := false
	for i := 0; i < len(data); i++ {
		b := data[i]
		if b < 0x80 {
			continue
		}

		hasNonASCII = true

		if b < 0xC2 || b > 0xF4 {
			return false
		}

		var size int
		if b < 0xE0 {
			size = 2
		} else if b < 0xF0 {
			size = 3
		} else {
			size = 4
		}

		if i+size > len(data) {
			return false
		}

		for j := 1; j < size; j++ {
			if data[i+j]&0xC0 != 0x80 {
				return false
			}
		}

		i += size - 1
	}

	return hasNonASCII || utf8.Valid(data)
}

func scoreASCII(data []byte) float64 {
	if isASCII(data) {
		return 1.0
	}
	return 0
}

func scoreUTF8(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.9
	}

	if isValidUTF8Sequence(data) {
		nonASCIICount := 0
		for _, b := range data {
			if b >= 0x80 {
				nonASCIICount++
			}
		}
		ratio := float64(nonASCIICount) / float64(len(data))
		if ratio > 0.8 {
			return 0.95
		}
		return 0.85
	}

	return 0
}

func scoreWindows1252(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.3
	}

	score := 0.0
	count := 0

	for _, b := range data {
		if b >= 0x80 && b <= 0x9F {
			score += 0.3
			count++
		} else if b >= 0xA0 && b <= 0xFF {
			score += 0.1
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreISO88591(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.3
	}

	hasControl := false
	for _, b := range data {
		if b >= 0x80 && b <= 0x9F {
			hasControl = true
			break
		}
	}

	if hasControl {
		return 0
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xA0 && b <= 0xFF {
			score += 0.1
		}
	}

	return score / float64(len(data))
}

func scoreISO88592(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if (b >= 0xA0 && b <= 0xBF) || (b >= 0xC0 && b <= 0xFF) {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreISO88595(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			score += 0.1
		}
	}

	return score / float64(len(data))
}

func scoreISO88596(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xA0 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreISO88597(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xA0 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreISO88598(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if (b >= 0xA0 && b <= 0xBE) || (b >= 0xE0 && b <= 0xFF) {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1250(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.3
	}

	score := 0.0
	count := 0

	for _, b := range data {
		if (b >= 0x80 && b <= 0x9F) || (b >= 0xC0 && b <= 0xFF) {
			score += 0.1
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreWindows1251(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.3
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			score += 0.12
		}
	}

	return math.Min(score/float64(len(data)), 0.8)
}

func scoreWindows1253(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0x80 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1254(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1255(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if (b >= 0x80 && b <= 0x9A) || (b >= 0xA0 && b <= 0xFF) {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1256(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0x80 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1257(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xA0 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreWindows1258(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0x80 && b <= 0xFF {
			score += 0.08
		}
	}

	return score / float64(len(data))
}

func scoreKOI8R(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			score += 0.1
		}
	}

	return score / float64(len(data))
}

func scoreKOI8U(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.2
	}

	score := 0.0
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			score += 0.1
		}
	}

	return score / float64(len(data))
}

func scoreUTF16LE(data []byte) float64 {
	if len(data) < 2 {
		return 0
	}

	if len(data)%2 != 0 {
		return 0
	}

	nullCount := 0
	for i := 1; i < len(data); i += 2 {
		if data[i] == 0 {
			nullCount++
		}
	}

	ratio := float64(nullCount) / float64(len(data) / 2)
	if ratio > 0.75 {
		return 0.8
	}

	return 0
}

func scoreUTF16BE(data []byte) float64 {
	if len(data) < 2 {
		return 0
	}

	if len(data)%2 != 0 {
		return 0
	}

	nullCount := 0
	for i := 0; i < len(data); i += 2 {
		if data[i] == 0 {
			nullCount++
		}
	}

	ratio := float64(nullCount) / float64(len(data) / 2)
	if ratio > 0.75 {
		return 0.8
	}

	return 0
}

func scoreShiftJIS(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if (b >= 0x81 && b <= 0x9F) || (b >= 0xE0 && b <= 0xEF) {
			if i+1 < len(data) {
				trail := data[i+1]
				if (trail >= 0x40 && trail <= 0x7E) || (trail >= 0x80 && trail <= 0xFC) {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreEUCJP(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0xA1 && b <= 0xFE {
			if i+1 < len(data) {
				trail := data[i+1]
				if trail >= 0xA1 && trail <= 0xFE {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreISO2022JP(data []byte) float64 {
	escapeCount := 0
	for i := 0; i < len(data)-2; i++ {
		if data[i] == 0x1B {
			if data[i+1] == '$' && data[i+2] == 'B' {
				escapeCount++
			}
		}
	}

	if escapeCount > 0 {
		return math.Min(float64(escapeCount)*0.2, 0.7)
	}

	return 0
}

func scoreGBK(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0x81 && b <= 0xFE {
			if i+1 < len(data) {
				trail := data[i+1]
				if (trail >= 0x40 && trail <= 0x7E) || (trail >= 0x80 && trail <= 0xFE) {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreGB18030(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0x81 && b <= 0xFE {
			if i+1 < len(data) {
				trail := data[i+1]
				if trail >= 0x30 && trail <= 0x39 {
					if i+3 < len(data) && data[i+2] >= 0x81 && data[i+2] <= 0xFE {
						score += 0.2
						count++
						i += 3
						continue
					}
				}

				if (trail >= 0x40 && trail <= 0x7E) || (trail >= 0x80 && trail <= 0xFE) {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreGB2312(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0xA1 && b <= 0xFE {
			if i+1 < len(data) {
				trail := data[i+1]
				if trail >= 0xA1 && trail <= 0xFE {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreBig5(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0xA1 && b <= 0xF9 {
			if i+1 < len(data) {
				trail := data[i+1]
				if (trail >= 0x40 && trail <= 0x7E) || (trail >= 0x80 && trail <= 0xFE) {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func scoreEUCKR(data []byte) float64 {
	if !hasNonASCIIBytes(data) {
		return 0.1
	}

	score := 0.0
	count := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= 0xA1 && b <= 0xFE {
			if i+1 < len(data) {
				trail := data[i+1]
				if trail >= 0xA1 && trail <= 0xFE {
					score += 0.15
					count++
					i++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return score / float64(len(data))
}

func hasNonASCIIBytes(data []byte) bool {
	for _, b := range data {
		if b >= 0x80 {
			return true
		}
	}
	return false
}

func NormalizeToUTF8(data []byte, detected EncodingResult) string {
	data = stripBOM(data, detected)

	switch detected.Encoding {
	case "ascii":
		return string(data)

	case "utf-8":
		return string(bytes.ToValidUTF8(data, []byte("\uFFFD")))

	case "utf-16le":
		return decodeWithFallback(data, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())

	case "utf-16be":
		return decodeWithFallback(data, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder())

	case "windows-1250":
		return decodeWithFallback(data, charmap.Windows1250.NewDecoder())

	case "windows-1251":
		return decodeWithFallback(data, charmap.Windows1251.NewDecoder())

	case "windows-1252":
		return decodeWithFallback(data, charmap.Windows1252.NewDecoder())

	case "windows-1253":
		return decodeWithFallback(data, charmap.Windows1253.NewDecoder())

	case "windows-1254":
		return decodeWithFallback(data, charmap.Windows1254.NewDecoder())

	case "windows-1255":
		return decodeWithFallback(data, charmap.Windows1255.NewDecoder())

	case "windows-1256":
		return decodeWithFallback(data, charmap.Windows1256.NewDecoder())

	case "windows-1257":
		return decodeWithFallback(data, charmap.Windows1257.NewDecoder())

	case "windows-1258":
		return decodeWithFallback(data, charmap.Windows1258.NewDecoder())

	case "iso-8859-1":
		return decodeWithFallback(data, charmap.ISO8859_1.NewDecoder())

	case "iso-8859-2":
		return decodeWithFallback(data, charmap.ISO8859_2.NewDecoder())

	case "iso-8859-5":
		return decodeWithFallback(data, charmap.ISO8859_5.NewDecoder())

	case "iso-8859-6":
		return decodeWithFallback(data, charmap.ISO8859_6.NewDecoder())

	case "iso-8859-7":
		return decodeWithFallback(data, charmap.ISO8859_7.NewDecoder())

	case "iso-8859-8":
		return decodeWithFallback(data, charmap.ISO8859_8.NewDecoder())

	case "koi8r":
		return decodeWithFallback(data, charmap.KOI8R.NewDecoder())

	case "koi8u":
		return decodeWithFallback(data, charmap.KOI8U.NewDecoder())

	case "shift-jis":
		return decodeWithFallback(data, japanese.ShiftJIS.NewDecoder())

	case "euc-jp":
		return decodeWithFallback(data, japanese.EUCJP.NewDecoder())

	case "iso-2022-jp":
		return decodeWithFallback(data, japanese.ISO2022JP.NewDecoder())

	case "gbk":
		return decodeWithFallback(data, simplifiedchinese.GBK.NewDecoder())

	case "gb18030":
		return decodeWithFallback(data, simplifiedchinese.GB18030.NewDecoder())

	case "gb2312":
		return decodeWithFallback(data, simplifiedchinese.HZGB2312.NewDecoder())

	case "big5":
		return decodeWithFallback(data, traditionalchinese.Big5.NewDecoder())

	case "euc-kr":
		return decodeWithFallback(data, korean.EUCKR.NewDecoder())

	default:
		return string(bytes.ToValidUTF8(data, []byte("\uFFFD")))
	}
}

func stripBOM(data []byte, detected EncodingResult) []byte {
	if !detected.HasBOM {
		return data
	}

	switch detected.Encoding {
	case "utf-8":
		if len(data) >= 3 && bytes.Equal(data[:3], []byte{0xEF, 0xBB, 0xBF}) {
			return data[3:]
		}

	case "utf-16le":
		if len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFF, 0xFE}) {
			return data[2:]
		}

	case "utf-16be":
		if len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFE, 0xFF}) {
			return data[2:]
		}
	}

	return data
}

func decodeWithFallback(data []byte, decoder *encoding.Decoder) string {
	if len(data) == 0 {
		return ""
	}

	reader := transform.NewReader(bytes.NewReader(data), decoder)
	result, err := io.ReadAll(reader)
	if err != nil {
		return string(bytes.ToValidUTF8(data, []byte("\uFFFD")))
	}

	return string(bytes.ToValidUTF8(result, []byte("\uFFFD")))
}

func ReadFileAsUTF8(path string) (content string, detected EncodingResult, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", EncodingResult{}, err
	}

	detected = DetectEncoding(data)
	content = NormalizeToUTF8(data, detected)
	return content, detected, nil
}

func ProbeFileEncoding(path string, maxProbe int) (EncodingResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return EncodingResult{}, err
	}
	defer file.Close()

	probe := make([]byte, maxProbe)
	n, err := file.Read(probe)
	if err != nil && err != io.EOF {
		return EncodingResult{}, err
	}

	return DetectEncoding(probe[:n]), nil
}
