package dat

// cp866High — символы CP866 с кодами 0x80..0xFF.
var cp866High = []rune("" +
	"АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" +
	"абвгдежзийклмноп" +
	"░▒▓│┤╡╢╖╕╣║╗╝╜╛┐└┴┬├─┼╞╟╚╔╩╦╠═╬╧╨╤╥╙╘╒╓╫╪┘┌█▄▌▐▀" +
	"рстуфхцчшщъыьэюя" +
	"ЁёЄєЇїЎў°∙·√№¤■ ")

var cp866Rev = func() map[rune]byte {
	m := make(map[rune]byte, len(cp866High))
	for i, r := range cp866High {
		m[r] = byte(0x80 + i)
	}
	return m
}()

// CP866ToRune переводит байт CP866 в символ Unicode.
func CP866ToRune(b byte) rune {
	if b < 0x80 {
		return rune(b)
	}
	return cp866High[b-0x80]
}

// RuneToCP866 переводит символ Unicode в CP866.
func RuneToCP866(r rune) (byte, bool) {
	if r < 0x80 {
		return byte(r), true
	}
	b, ok := cp866Rev[r]
	return b, ok
}
