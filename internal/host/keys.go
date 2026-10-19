package host

// Key — служебная клавиша PC: скан-код и ASCII-код в буфере BIOS.
type Key struct {
	Scan, ASCII byte
}

// Служебные клавиши по именам (для сценариев и раскладки окна).
var KeyByName = map[string]Key{
	"esc":   {0x01, 27},
	"enter": {0x1C, 13},
	"bksp":  {0x0E, 8},
	"tab":   {0x0F, 9},
	"space": {0x39, ' '},
	"up":    {0x48, 0},
	"down":  {0x50, 0},
	"left":  {0x4B, 0},
	"right": {0x4D, 0},
	"home":  {0x47, 0},
	"end":   {0x4F, 0},
	"pgup":  {0x49, 0},
	"pgdn":  {0x51, 0},
	"ins":   {0x52, 0},
	"del":   {0x53, 0},
	"f1":    {0x3B, 0},
	"f2":    {0x3C, 0},
	"f3":    {0x3D, 0},
	"f4":    {0x3E, 0},
	"f5":    {0x3F, 0},
	"f6":    {0x40, 0},
}

// scanOfChar — скан-коды символьных клавиш (для полноты буфера BIOS).
var scanOfChar = map[byte]byte{
	'1': 0x02, '2': 0x03, '3': 0x04, '4': 0x05, '5': 0x06, '6': 0x07, '7': 0x08, '8': 0x09, '9': 0x0A, '0': 0x0B,
	'q': 0x10, 'w': 0x11, 'e': 0x12, 'r': 0x13, 't': 0x14, 'y': 0x15, 'u': 0x16, 'i': 0x17, 'o': 0x18, 'p': 0x19,
	'a': 0x1E, 's': 0x1F, 'd': 0x20, 'f': 0x21, 'g': 0x22, 'h': 0x23, 'j': 0x24, 'k': 0x25, 'l': 0x26,
	'z': 0x2C, 'x': 0x2D, 'c': 0x2E, 'v': 0x2F, 'b': 0x30, 'n': 0x31, 'm': 0x32, ' ': 0x39,
}

// CharKey возвращает клавишу для символа CP866.
func CharKey(c byte) Key {
	lc := c
	if lc >= 'A' && lc <= 'Z' {
		lc += 'a' - 'A'
	}
	return Key{Scan: scanOfChar[lc], ASCII: c}
}
