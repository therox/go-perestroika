// Перестройка — порт аркадной игры Н. Скрипкина (Locis, 1990).
//
// Графика, звук и шрифт берутся из файла данных игры TOPPLER.DAT, который
// в программу не входит: он ищется рядом с программой, в текущем каталоге
// и в каталоге настроек игры, либо указывается флагом -dat.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/host"
)

func main() {
	datPath := flag.String("dat", "", "путь к файлу данных TOPPLER.DAT")
	speed := flag.Float64("speed", 1, "скорость игры (больше — быстрее)")
	fullscreen := flag.Bool("fullscreen", false, "полноэкранный режим")
	volume := flag.Float64("volume", 0.7, "громкость 0..1")
	audioDelay := flag.Int("audio-delay", -1, "задержка картинки под звук, мс (для Bluetooth-наушников; по умолчанию — сохранённая)")
	scoresPath := flag.String("scores", host.DefaultHiScorePath(), "файл таблицы рекордов")
	script := flag.String("script", "", "автономный прогон по сценарию (для отладки)")
	outDir := flag.String("out", ".", "каталог для снимков автономного прогона")
	wavPath := flag.String("wav", "", "записать звук автономного прогона в WAV")
	fixedRnd := flag.Int("debug-rnd", 0, "отладка: фиксированное значение часового генератора (детерминированная игра)")
	startLevel := flag.Int("debug-level", 0, "отладка: начать с указанного этапа")
	stateLog := flag.String("debug-state", "", "отладка: писать состояние игры в начале каждого тика в файл")
	capture := flag.String("debug-capture", "", "отладка: сохранить кадр окна через 3 с в PNG и выйти")
	flag.Parse()

	data, err := loadData(*datPath)
	store := &host.HiScoreFile{Path: *scoresPath}

	if *script != "" {
		if err != nil {
			fail(err)
		}
		dbg := debugOptions{fixedRnd: *fixedRnd, startLevel: *startLevel, stateLog: *stateLog}
		if err := runScript(data, store.Load(data.HiScores), *script, *outDir, *wavPath, *speed, dbg); err != nil {
			fail(err)
		}
		return
	}
	// без файла данных окно всё равно открывается: в нём объясняется, где его взять
	if err := runWindow(data, err, store, *speed, *volume, *fullscreen, *capture, *audioDelay); err != nil {
		fail(err)
	}
}

// loadData находит и читает файл данных.
func loadData(path string) (*dat.Data, error) {
	if path == "" {
		p, err := host.FindDataFile()
		if err != nil {
			return nil, err
		}
		path = p
	}
	d, err := host.LoadDataFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "perestroika:", err)
	os.Exit(1)
}
