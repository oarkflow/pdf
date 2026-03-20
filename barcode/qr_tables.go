package barcode

// QR Code version capacity and error correction tables.

// ECLevel represents the error correction level.
type ECLevel int

const (
	ECLow      ECLevel = iota // L - 7% recovery
	ECMedium                  // M - 15%
	ECQuartile                // Q - 25%
	ECHigh                    // H - 30%
)

// qrVersionInfo holds capacity and EC info for one version+EC level combination.
type qrVersionInfo struct {
	TotalCodewords int
	ECPerBlock     int
	Group1Blocks   int
	Group1Data     int
	Group2Blocks   int
	Group2Data     int
}

// qrCapacity[version-1][ecLevel] = data capacity in bytes (after EC).
// qrECInfo[version-1][ecLevel] = error correction block info.

var qrECInfo [40][4]qrVersionInfo

// qrDataCapacity[version-1][ecLevel] = total data codewords
var qrDataCapacity [40][4]int

// qrAlignmentPatterns[version] = center coordinates of alignment patterns
var qrAlignmentPatterns [41][]int

// qrVersionCapacity[version-1][ecLevel][mode] = max chars
// mode: 0=numeric, 1=alphanumeric, 2=byte
var qrVersionCapacity [40][4][3]int

func init() {
	initQRECInfo()
	initAlignmentPatterns()
	initVersionCapacity()
}

func initQRECInfo() {
	// version, ecLevel, totalCW, ecPerBlock, g1Blocks, g1Data, g2Blocks, g2Data
	type entry struct {
		ver    int
		ec     ECLevel
		total  int
		ecPB   int
		g1B    int
		g1D    int
		g2B    int
		g2D    int
	}

	entries := []entry{
		// Version 1
		{1, ECLow, 26, 7, 1, 19, 0, 0},
		{1, ECMedium, 26, 10, 1, 16, 0, 0},
		{1, ECQuartile, 26, 13, 1, 13, 0, 0},
		{1, ECHigh, 26, 17, 1, 9, 0, 0},
		// Version 2
		{2, ECLow, 44, 10, 1, 34, 0, 0},
		{2, ECMedium, 44, 16, 1, 28, 0, 0},
		{2, ECQuartile, 44, 22, 1, 22, 0, 0},
		{2, ECHigh, 44, 28, 1, 16, 0, 0},
		// Version 3
		{3, ECLow, 70, 15, 1, 55, 0, 0},
		{3, ECMedium, 70, 26, 1, 44, 0, 0},
		{3, ECQuartile, 70, 18, 2, 17, 0, 0},
		{3, ECHigh, 70, 22, 2, 13, 0, 0},
		// Version 4
		{4, ECLow, 100, 20, 1, 80, 0, 0},
		{4, ECMedium, 100, 18, 2, 32, 0, 0},
		{4, ECQuartile, 100, 26, 2, 24, 0, 0},
		{4, ECHigh, 100, 16, 4, 9, 0, 0},
		// Version 5
		{5, ECLow, 134, 26, 1, 108, 0, 0},
		{5, ECMedium, 134, 24, 2, 43, 0, 0},
		{5, ECQuartile, 134, 18, 2, 15, 2, 16},
		{5, ECHigh, 134, 22, 2, 11, 2, 12},
		// Version 6
		{6, ECLow, 172, 18, 2, 68, 0, 0},
		{6, ECMedium, 172, 16, 4, 27, 0, 0},
		{6, ECQuartile, 172, 24, 4, 19, 0, 0},
		{6, ECHigh, 172, 28, 4, 15, 0, 0},
		// Version 7
		{7, ECLow, 196, 20, 2, 78, 0, 0},
		{7, ECMedium, 196, 18, 4, 31, 0, 0},
		{7, ECQuartile, 196, 18, 2, 14, 4, 15},
		{7, ECHigh, 196, 26, 4, 13, 1, 14},
		// Version 8
		{8, ECLow, 242, 24, 2, 97, 0, 0},
		{8, ECMedium, 242, 22, 2, 38, 2, 39},
		{8, ECQuartile, 242, 22, 4, 18, 2, 19},
		{8, ECHigh, 242, 26, 4, 14, 2, 15},
		// Version 9
		{9, ECLow, 292, 30, 2, 116, 0, 0},
		{9, ECMedium, 292, 22, 3, 36, 2, 37},
		{9, ECQuartile, 292, 20, 4, 16, 4, 17},
		{9, ECHigh, 292, 24, 4, 12, 4, 13},
		// Version 10
		{10, ECLow, 346, 18, 2, 68, 2, 69},
		{10, ECMedium, 346, 26, 4, 43, 1, 44},
		{10, ECQuartile, 346, 24, 6, 19, 2, 20},
		{10, ECHigh, 346, 28, 6, 15, 2, 16},
		// Version 11
		{11, ECLow, 404, 20, 4, 81, 0, 0},
		{11, ECMedium, 404, 30, 1, 50, 4, 51},
		{11, ECQuartile, 404, 28, 4, 22, 4, 23},
		{11, ECHigh, 404, 24, 3, 12, 8, 13},
		// Version 12
		{12, ECLow, 466, 24, 2, 92, 2, 93},
		{12, ECMedium, 466, 22, 6, 36, 2, 37},
		{12, ECQuartile, 466, 26, 4, 20, 6, 21},
		{12, ECHigh, 466, 28, 7, 14, 4, 15},
		// Version 13
		{13, ECLow, 532, 26, 4, 107, 0, 0},
		{13, ECMedium, 532, 22, 8, 37, 1, 38},
		{13, ECQuartile, 532, 24, 8, 20, 4, 21},
		{13, ECHigh, 532, 22, 12, 11, 4, 12},
		// Version 14
		{14, ECLow, 581, 30, 3, 115, 1, 116},
		{14, ECMedium, 581, 24, 4, 40, 5, 41},
		{14, ECQuartile, 581, 20, 11, 16, 5, 17},
		{14, ECHigh, 581, 24, 11, 12, 5, 13},
		// Version 15
		{15, ECLow, 655, 22, 5, 87, 1, 88},
		{15, ECMedium, 655, 24, 5, 41, 5, 42},
		{15, ECQuartile, 655, 30, 5, 24, 7, 25},
		{15, ECHigh, 655, 24, 11, 12, 7, 13},
		// Version 16
		{16, ECLow, 733, 24, 5, 98, 1, 99},
		{16, ECMedium, 733, 28, 7, 45, 3, 46},
		{16, ECQuartile, 733, 24, 15, 19, 2, 20},
		{16, ECHigh, 733, 30, 3, 15, 13, 16},
		// Version 17
		{17, ECLow, 815, 28, 1, 107, 5, 108},
		{17, ECMedium, 815, 28, 10, 46, 1, 47},
		{17, ECQuartile, 815, 28, 1, 22, 15, 23},
		{17, ECHigh, 815, 28, 2, 14, 17, 15},
		// Version 18
		{18, ECLow, 901, 30, 5, 120, 1, 121},
		{18, ECMedium, 901, 26, 9, 43, 4, 44},
		{18, ECQuartile, 901, 28, 17, 22, 1, 23},
		{18, ECHigh, 901, 28, 2, 14, 19, 15},
		// Version 19
		{19, ECLow, 991, 28, 3, 113, 4, 114},
		{19, ECMedium, 991, 26, 3, 44, 11, 45},
		{19, ECQuartile, 991, 26, 17, 21, 4, 22},
		{19, ECHigh, 991, 26, 9, 13, 16, 14},
		// Version 20
		{20, ECLow, 1085, 28, 3, 107, 5, 108},
		{20, ECMedium, 1085, 26, 3, 41, 13, 42},
		{20, ECQuartile, 1085, 30, 15, 24, 5, 25},
		{20, ECHigh, 1085, 28, 15, 15, 10, 16},
		// Version 21
		{21, ECLow, 1156, 28, 4, 116, 4, 117},
		{21, ECMedium, 1156, 26, 17, 42, 0, 0},
		{21, ECQuartile, 1156, 28, 17, 22, 6, 23},
		{21, ECHigh, 1156, 30, 19, 16, 6, 17},
		// Version 22
		{22, ECLow, 1258, 28, 2, 111, 7, 112},
		{22, ECMedium, 1258, 28, 17, 46, 0, 0},
		{22, ECQuartile, 1258, 30, 7, 24, 16, 25},
		{22, ECHigh, 1258, 24, 34, 13, 0, 0},
		// Version 23
		{23, ECLow, 1364, 30, 4, 121, 5, 122},
		{23, ECMedium, 1364, 28, 4, 47, 14, 48},
		{23, ECQuartile, 1364, 30, 11, 24, 14, 25},
		{23, ECHigh, 1364, 30, 16, 15, 14, 16},
		// Version 24
		{24, ECLow, 1474, 30, 6, 117, 4, 118},
		{24, ECMedium, 1474, 28, 6, 45, 14, 46},
		{24, ECQuartile, 1474, 30, 11, 24, 16, 25},
		{24, ECHigh, 1474, 30, 30, 16, 2, 17},
		// Version 25
		{25, ECLow, 1588, 26, 8, 106, 4, 107},
		{25, ECMedium, 1588, 28, 8, 47, 13, 48},
		{25, ECQuartile, 1588, 30, 7, 24, 22, 25},
		{25, ECHigh, 1588, 30, 22, 15, 13, 16},
		// Version 26
		{26, ECLow, 1706, 28, 10, 114, 2, 115},
		{26, ECMedium, 1706, 28, 19, 46, 4, 47},
		{26, ECQuartile, 1706, 28, 28, 22, 6, 23},
		{26, ECHigh, 1706, 30, 33, 16, 4, 17},
		// Version 27
		{27, ECLow, 1828, 30, 8, 122, 4, 123},
		{27, ECMedium, 1828, 28, 22, 45, 3, 46},
		{27, ECQuartile, 1828, 30, 8, 23, 26, 24},
		{27, ECHigh, 1828, 30, 12, 15, 28, 16},
		// Version 28
		{28, ECLow, 1921, 30, 3, 117, 10, 118},
		{28, ECMedium, 1921, 28, 3, 45, 23, 46},
		{28, ECQuartile, 1921, 30, 4, 24, 31, 25},
		{28, ECHigh, 1921, 30, 11, 15, 31, 16},
		// Version 29
		{29, ECLow, 2051, 30, 7, 116, 7, 117},
		{29, ECMedium, 2051, 28, 21, 45, 7, 46},
		{29, ECQuartile, 2051, 30, 1, 23, 37, 24},
		{29, ECHigh, 2051, 30, 19, 15, 26, 16},
		// Version 30
		{30, ECLow, 2185, 30, 5, 115, 10, 116},
		{30, ECMedium, 2185, 28, 19, 47, 10, 48},
		{30, ECQuartile, 2185, 30, 15, 24, 25, 25},
		{30, ECHigh, 2185, 30, 23, 15, 25, 16},
		// Version 31
		{31, ECLow, 2323, 30, 13, 115, 3, 116},
		{31, ECMedium, 2323, 28, 2, 46, 29, 47},
		{31, ECQuartile, 2323, 30, 42, 24, 1, 25},
		{31, ECHigh, 2323, 30, 23, 15, 28, 16},
		// Version 32
		{32, ECLow, 2465, 30, 17, 115, 0, 0},
		{32, ECMedium, 2465, 28, 10, 46, 23, 47},
		{32, ECQuartile, 2465, 30, 10, 24, 35, 25},
		{32, ECHigh, 2465, 30, 19, 15, 35, 16},
		// Version 33
		{33, ECLow, 2611, 30, 17, 115, 1, 116},
		{33, ECMedium, 2611, 28, 14, 46, 21, 47},
		{33, ECQuartile, 2611, 30, 29, 24, 19, 25},
		{33, ECHigh, 2611, 30, 11, 15, 46, 16},
		// Version 34
		{34, ECLow, 2761, 30, 13, 115, 6, 116},
		{34, ECMedium, 2761, 28, 14, 46, 23, 47},
		{34, ECQuartile, 2761, 30, 44, 24, 7, 25},
		{34, ECHigh, 2761, 30, 59, 16, 1, 17},
		// Version 35
		{35, ECLow, 2876, 30, 12, 121, 7, 122},
		{35, ECMedium, 2876, 28, 12, 47, 26, 48},
		{35, ECQuartile, 2876, 30, 39, 24, 14, 25},
		{35, ECHigh, 2876, 30, 22, 15, 41, 16},
		// Version 36
		{36, ECLow, 3034, 30, 6, 121, 14, 122},
		{36, ECMedium, 3034, 28, 6, 47, 34, 48},
		{36, ECQuartile, 3034, 30, 46, 24, 10, 25},
		{36, ECHigh, 3034, 30, 2, 15, 64, 16},
		// Version 37
		{37, ECLow, 3196, 30, 17, 122, 4, 123},
		{37, ECMedium, 3196, 28, 29, 46, 14, 47},
		{37, ECQuartile, 3196, 30, 49, 24, 10, 25},
		{37, ECHigh, 3196, 30, 24, 15, 46, 16},
		// Version 38
		{38, ECLow, 3362, 30, 4, 122, 18, 123},
		{38, ECMedium, 3362, 28, 13, 46, 32, 47},
		{38, ECQuartile, 3362, 30, 48, 24, 14, 25},
		{38, ECHigh, 3362, 30, 42, 15, 32, 16},
		// Version 39
		{39, ECLow, 3532, 30, 20, 117, 4, 118},
		{39, ECMedium, 3532, 28, 40, 47, 7, 48},
		{39, ECQuartile, 3532, 30, 43, 24, 22, 25},
		{39, ECHigh, 3532, 30, 10, 15, 67, 16},
		// Version 40
		{40, ECLow, 3706, 30, 19, 118, 6, 119},
		{40, ECMedium, 3706, 28, 18, 47, 31, 48},
		{40, ECQuartile, 3706, 30, 34, 24, 34, 25},
		{40, ECHigh, 3706, 30, 20, 15, 61, 16},
	}

	for _, e := range entries {
		qrECInfo[e.ver-1][e.ec] = qrVersionInfo{
			TotalCodewords: e.total,
			ECPerBlock:     e.ecPB,
			Group1Blocks:   e.g1B,
			Group1Data:     e.g1D,
			Group2Blocks:   e.g2B,
			Group2Data:     e.g2D,
		}
		qrDataCapacity[e.ver-1][e.ec] = e.g1B*e.g1D + e.g2B*e.g2D
	}
}

func initAlignmentPatterns() {
	// Alignment pattern center positions per version
	ap := [41][]int{
		{},   // version 0 unused
		{},   // version 1 - no alignment patterns
		{6, 18},
		{6, 22},
		{6, 26},
		{6, 30},
		{6, 34},
		{6, 22, 38},
		{6, 24, 42},
		{6, 26, 46},
		{6, 28, 50},
		{6, 30, 54},
		{6, 32, 58},
		{6, 34, 62},
		{6, 26, 46, 66},
		{6, 26, 48, 70},
		{6, 26, 50, 74},
		{6, 30, 54, 78},
		{6, 30, 56, 82},
		{6, 30, 58, 86},
		{6, 34, 62, 90},
		{6, 28, 50, 72, 94},
		{6, 26, 50, 74, 98},
		{6, 30, 54, 78, 102},
		{6, 28, 54, 80, 106},
		{6, 32, 58, 84, 110},
		{6, 30, 58, 86, 114},
		{6, 34, 62, 90, 118},
		{6, 26, 50, 74, 98, 122},
		{6, 30, 54, 78, 102, 126},
		{6, 26, 52, 78, 104, 130},
		{6, 30, 56, 82, 108, 134},
		{6, 34, 60, 86, 112, 138},
		{6, 30, 58, 86, 114, 142},
		{6, 34, 62, 90, 118, 146},
		{6, 30, 54, 78, 102, 126, 150},
		{6, 24, 50, 76, 102, 128, 154},
		{6, 28, 54, 80, 106, 132, 158},
		{6, 32, 58, 84, 110, 136, 162},
		{6, 26, 54, 82, 110, 138, 166},
		{6, 30, 58, 86, 114, 142, 170},
	}
	qrAlignmentPatterns = ap
}

func initVersionCapacity() {
	// Character count indicator bit lengths per version range per mode
	// mode: 0=numeric, 1=alphanumeric, 2=byte
	// Computed from data capacity in bytes and encoding overhead.
	for v := 1; v <= 40; v++ {
		for ec := ECLow; ec <= ECHigh; ec++ {
			dataCW := qrDataCapacity[v-1][ec]
			dataBits := dataCW * 8

			// Character count indicator sizes
			var numBits, alphaBits, byteBits int
			if v <= 9 {
				numBits, alphaBits, byteBits = 10, 9, 8
			} else if v <= 26 {
				numBits, alphaBits, byteBits = 12, 11, 16
			} else {
				numBits, alphaBits, byteBits = 14, 13, 16
			}

			// Numeric: 4 mode + numBits count + ceil(n/3)*10 (groups of 3->10bits, rem2->7, rem1->4)
			// Solve for max n
			available := dataBits - 4 - numBits
			if available > 0 {
				// 3 digits = 10 bits; approx
				n := (available * 3) / 10
				for numericBitLen(n+1) <= available {
					n++
				}
				for n > 0 && numericBitLen(n) > available {
					n--
				}
				qrVersionCapacity[v-1][ec][0] = n
			}

			// Alphanumeric: 4 mode + alphaBits count + ceil(n/2)*11 ...
			available = dataBits - 4 - alphaBits
			if available > 0 {
				n := (available * 2) / 11
				for alphanumericBitLen(n+1) <= available {
					n++
				}
				for n > 0 && alphanumericBitLen(n) > available {
					n--
				}
				qrVersionCapacity[v-1][ec][1] = n
			}

			// Byte: 4 mode + byteBits count + n*8
			available = dataBits - 4 - byteBits
			if available > 0 {
				n := available / 8
				qrVersionCapacity[v-1][ec][2] = n
			}
		}
	}
}

func numericBitLen(n int) int {
	full := n / 3
	rem := n % 3
	bits := full * 10
	if rem == 2 {
		bits += 7
	} else if rem == 1 {
		bits += 4
	}
	return bits
}

func alphanumericBitLen(n int) int {
	full := n / 2
	rem := n % 2
	return full*11 + rem*6
}

// Format information: 15-bit BCH code for EC level + mask pattern
// Precomputed format info bits (with mask 101010000010010 applied)
var qrFormatInfo = [4][8]uint32{
	// ECLow (L=01)
	{0x77C4, 0x72F3, 0x7DAA, 0x789D, 0x662F, 0x6318, 0x6C41, 0x6976},
	// ECMedium (M=00)
	{0x5412, 0x5125, 0x5E7C, 0x5B4B, 0x45F9, 0x40CE, 0x4F97, 0x4AA0},
	// ECQuartile (Q=11)
	{0x355F, 0x3068, 0x3F31, 0x3A06, 0x24B4, 0x2183, 0x2EDA, 0x2BED},
	// ECHigh (H=10)
	{0x1689, 0x13BE, 0x1CE7, 0x19D0, 0x0762, 0x0255, 0x0D0C, 0x083B},
}

// Version information: 18-bit BCH code for versions 7-40
var qrVersionInfoBits [41]uint32

func init() {
	// Precomputed version info (18-bit) for versions 7-40
	vi := [34]uint32{
		0x07C94, 0x085BC, 0x09A99, 0x0A4D3, 0x0BBF6, 0x0C762, 0x0D847, 0x0E60D,
		0x0F928, 0x10B78, 0x1145D, 0x12A17, 0x13532, 0x149A6, 0x15683, 0x168C9,
		0x177EC, 0x18EC4, 0x191E1, 0x1AFAB, 0x1B08E, 0x1CC1A, 0x1D33F, 0x1ED75,
		0x1F250, 0x209D5, 0x216F0, 0x228BA, 0x2379F, 0x24B0B, 0x2542E, 0x26A64,
		0x27541, 0x28C69,
	}
	for i, v := range vi {
		qrVersionInfoBits[i+7] = v
	}
}

// Alphanumeric character set for QR Code
const qrAlphanumericChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

var qrAlphanumericMap [128]int

func init() {
	for i := range qrAlphanumericMap {
		qrAlphanumericMap[i] = -1
	}
	for i, c := range qrAlphanumericChars {
		qrAlphanumericMap[c] = i
	}
}
