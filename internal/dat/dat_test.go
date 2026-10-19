package dat

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func load(t *testing.T) []byte {
	t.Helper()
	path := os.Getenv("PERESTROIKA_DAT")
	if path == "" {
		path = "../../Source/TOPPLER.DAT"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skip("нет файла данных игры (задайте PERESTROIKA_DAT):", err)
	}
	return raw
}

func TestParse(t *testing.T) {
	d, err := Parse(load(t))
	if err != nil {
		t.Fatal(err)
	}
	// длины звуков в 256-байтных страницах
	for i, pages := range []int{9, 28, 17, 13, 30} {
		if len(d.Sounds[i]) != pages*256 {
			t.Errorf("звук %d: %d байт, ожидалось %d страниц", i, len(d.Sounds[i]), pages)
		}
	}
	if len(d.Intro) != 617*256 {
		t.Errorf("заставка: %d байт", len(d.Intro))
	}
	if d.Font.MaxIndex() != 94 {
		t.Errorf("шрифт: %d символов", d.Font.MaxIndex()+1)
	}
	// «B» в раскладке ЙЦУКЕН — «И»
	if n := len(d.Font.Glyph('B' - 0x21)); n != 5 {
		t.Errorf("глиф «И»: %d штрихов", n)
	}
	if d.HiScores[0].Score != 196250 || d.HiScores[9].Score != 67050 {
		t.Errorf("рекорды: %d .. %d", d.HiScores[0].Score, d.HiScores[9].Score)
	}
}

func TestHiScoresRoundTrip(t *testing.T) {
	raw := load(t)
	tail := raw[offScores : offScores+lenScores]
	if got := EncodeHiScores(ParseHiScores(tail)); !bytes.Equal(got, tail) {
		t.Fatal("таблица рекордов искажается при перезаписи")
	}
}

// Посторонний файл отвергается с понятной ошибкой.
func TestParseRejectsOtherFiles(t *testing.T) {
	for _, raw := range [][]byte{nil, make([]byte, totalSize)} {
		if _, err := Parse(raw); !errors.Is(err, ErrFormat) {
			t.Fatalf("ожидалась ErrFormat, получено %v", err)
		}
	}
}

func TestCP866(t *testing.T) {
	if len(cp866High) != 128 {
		t.Fatalf("таблица CP866: %d символов", len(cp866High))
	}
	for b := 0; b < 256; b++ {
		if got, ok := RuneToCP866(CP866ToRune(byte(b))); !ok || got != byte(b) {
			t.Fatalf("CP866 0x%02x не переводится обратно", b)
		}
	}
}
