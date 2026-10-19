package game

import (
	"os"
	"testing"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/gfx"
	"github.com/therox/go-perestroika/internal/host"
	"github.com/therox/go-perestroika/internal/sfx"
)

// loadData читает файл данных игры: путь из PERESTROIKA_DAT или Source/TOPPLER.DAT.
func loadData(t *testing.T) *dat.Data {
	t.Helper()
	path := os.Getenv("PERESTROIKA_DAT")
	if path == "" {
		path = "../../Source/TOPPLER.DAT"
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

func TestTurboCRand(t *testing.T) {
	g := &Game{rng: 1}
	// первые значения rand() Turbo C 2.0 без srand()
	want := []int{346, 130, 10982, 1090, 11656}
	for i, w := range want {
		if v := g.rand(); v != w {
			t.Fatalf("rand #%d = %d, ожидалось %d", i, v, w)
		}
	}
}

func TestHiScoreEntry(t *testing.T) {
	data := loadData(t)
	kb := &host.Keyboard{}
	var evs []host.ScriptEvent
	at := 1.0
	for _, c := range []byte("TeSt") {
		c := c
		evs = append(evs, host.ScriptEvent{At: at, Do: func() { kb.Press(0, c) }})
		at += 0.2
	}
	evs = append(evs, host.ScriptEvent{At: at, Do: func() { kb.Press(0x1C, 13) }})
	clk := host.NewScriptClock(evs)
	g := New(gfx.NewScreen(), clk, kb, sfx.NewOfflineMixer(), data, nil, data.HiScores, 0)
	g.score = 100000
	if !g.checkHiScore() {
		t.Fatal("результат 100000 должен попасть в таблицу")
	}
	got := g.hiScores[3]
	if got.Score != 100000 || string(got.Name[:8]) != "test    " {
		t.Fatalf("запись 4: %d %q", got.Score, got.Name)
	}
	if g.hiScores[2].Score != 103700 || g.hiScores[4].Score != 97300 {
		t.Fatalf("таблица сдвинута неверно: %+v", g.hiScores)
	}
	if g.hiScores[9].Score != data.HiScores[8].Score {
		t.Fatal("последняя запись должна вытесняться")
	}
}

func TestLevelGeometry(t *testing.T) {
	data := loadData(t)
	clk := host.NewScriptClock(nil)
	g := New(gfx.NewScreen(), clk, &host.Keyboard{}, sfx.NewOfflineMixer(), data, nil, data.HiScores, 0)
	// значения сверены с памятью оригинальной программы на 1-м этапе
	g.cols, g.level = 6, 1
	if n := g.genLevel(); n != 28 {
		t.Fatalf("листьев %d, ожидалось 28", n)
	}
	if g.r != 35 || g.x0 != 70 || g.xMax != 568 || g.yTop != 78 || g.yBot != 294 || g.colStep != 83 || g.rowStep != 72 {
		t.Fatalf("геометрия: r=%d x=%d..%d y=%d..%d шаг %d/%d", g.r, g.x0, g.xMax, g.yTop, g.yBot, g.colStep, g.rowStep)
	}
	// самый мелкий этап
	g.cols = 13
	if n := g.genLevel(); n != 14*8 {
		t.Fatalf("на 13 столбцах листьев %d, ожидалось 112", n)
	}
}
