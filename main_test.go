package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/game"
	"github.com/therox/go-perestroika/internal/gfx"
	"github.com/therox/go-perestroika/internal/host"
	"github.com/therox/go-perestroika/internal/sfx"
)

// Эталонные хеши экранов (RGB-байты видимой страницы). Заставка попиксельно
// совпадает с оригиналом; главный экран — тоже, кроме начертания шрифта 8x8
// в таблице рекордов (используется свободный шрифт unscii-8).
const (
	titleHash = "05ad005267a5e1ec366add4739d0a993179dfd41049e90950d1a681c358f3107"
	mainHash  = "c858f5b3fbfd36637d98dde5f39e374a89f5675500a4d58fde77e8edbcb7e8a3"
)

// testData читает файл данных игры: путь из PERESTROIKA_DAT или Source/TOPPLER.DAT.
// Без файла тест пропускается: данные игры не входят в проект.
func testData(t *testing.T) *dat.Data {
	t.Helper()
	path := os.Getenv("PERESTROIKA_DAT")
	if path == "" {
		path = "Source/TOPPLER.DAT"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skip("нет файла данных игры (задайте PERESTROIKA_DAT):", err)
	}
	d, err := dat.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

type harness struct {
	t      *testing.T
	scr    *gfx.Screen
	kb     *host.Keyboard
	g      *game.Game
	clk    *host.ScriptClock
	shots  map[string]string
	ended  bool
	stored *memStore
}

type memStore struct{ saved [][10]dat.HiScore }

func (m *memStore) SaveHiScores(hs [10]dat.HiScore) error {
	m.saved = append(m.saved, hs)
	return nil
}

type step struct {
	at  float64
	key string
	// shot — имя снимка видимой страницы
	shot string
}

func newHarness(t *testing.T, steps []step, quitAt float64) *harness {
	t.Helper()
	data := testData(t)
	h := &harness{t: t, scr: gfx.NewScreen(), kb: &host.Keyboard{}, shots: map[string]string{}, stored: &memStore{}}
	var evs []host.ScriptEvent
	for _, s := range steps {
		s := s
		evs = append(evs, host.ScriptEvent{At: s.at, Do: func() {
			if s.key != "" {
				k, ok := host.KeyByName[s.key]
				if !ok {
					k = host.CharKey(s.key[0])
				}
				h.kb.Press(k.Scan, k.ASCII)
			}
			if s.shot != "" {
				h.shots[s.shot] = rgbHash(h.scr)
			}
		}})
	}
	evs = append(evs, host.ScriptEvent{At: quitAt, Do: func() { panic(errQuit) }})
	h.clk = host.NewScriptClock(evs)
	h.g = game.New(h.scr, h.clk, h.kb, sfx.NewOfflineMixer(), data, h.stored, data.HiScores, 12*3600)
	return h
}

func (h *harness) run() {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok && errors.Is(e, errQuit) {
				return
			}
			panic(r)
		}
	}()
	h.g.Run()
	h.ended = true
}

func rgbHash(scr *gfx.Screen) string {
	img := scr.Snapshot(0)
	sum := sha256.New()
	for i := 0; i < len(img.Pix); i += 4 {
		sum.Write(img.Pix[i : i+3])
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// Заставка и главный экран совпадают с эталоном пиксель в пиксель.
func TestScreensMatchOriginal(t *testing.T) {
	h := newHarness(t, []step{
		{at: 1, shot: "title"},
		{at: 2, key: "space"},
		{at: 4, shot: "main"},
	}, 5)
	h.run()
	if h.shots["title"] != titleHash {
		t.Errorf("заставка отличается: %s", h.shots["title"])
	}
	if h.shots["main"] != mainHash {
		t.Errorf("главный экран отличается: %s", h.shots["main"])
	}
}

// Неподвижный Демократ трижды тонет; на вопрос «Будем ещё
// перестраиваться?» ответ «0» завершает игру.
func TestGameOverAndQuit(t *testing.T) {
	h := newHarness(t, []step{
		{at: 1, key: "space"},
		{at: 3, key: "space"},
		{at: 150, key: "space"}, // закрыть статистику
		{at: 160, key: "0"},
	}, 600)
	h.run()
	if !h.ended {
		t.Fatal("игра не завершилась по ответу «0»")
	}
	if len(h.stored.saved) != 0 {
		t.Error("нулевой результат не должен попадать в таблицу рекордов")
	}
}

// Файл данных находится в перетащенной папке без учёта регистра имени.
func TestFindDropped(t *testing.T) {
	fsys := fstest.MapFS{
		"game/readme.txt":      {Data: []byte("x")},
		"game/sub/toppler.dat": {Data: []byte("data")},
	}
	b, err := findDropped(fsys)
	if err != nil || string(b) != "data" {
		t.Fatalf("findDropped: %q, %v", b, err)
	}
	if _, err := findDropped(fstest.MapFS{"a.txt": {Data: []byte("x")}}); !errors.Is(err, host.ErrNoData) {
		t.Fatalf("без файла данных: %v", err)
	}
}
