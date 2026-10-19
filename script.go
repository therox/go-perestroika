package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/game"
	"github.com/therox/go-perestroika/internal/gfx"
	"github.com/therox/go-perestroika/internal/host"
	"github.com/therox/go-perestroika/internal/sfx"
)

var errQuit = errors.New("quit")

// debugOptions — отладочные параметры автономного прогона.
type debugOptions struct {
	fixedRnd   int
	startLevel int
	stateLog   string
	tickKeys   map[int]string // номер тика -> клавиша
}

// runScript выполняет игру без окна в виртуальном времени по сценарию.
//
// Формат строк сценария: «<время, с> <команда> [аргументы]», где команды:
// key <имя|символ> — нажатие; shot <файл> [страница] — снимок экрана;
// tickkey <номер тика> <клавиша> — нажатие в начале тика; quit.
// Строки tickkey используют поле времени только для порядка.
func runScript(data *dat.Data, scores [10]dat.HiScore, path, outDir, wavPath string, speed float64, dbg debugOptions) error {
	scr := gfx.NewScreen()
	kb := &host.Keyboard{}
	mixer := sfx.NewOfflineMixer()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var events []host.ScriptEvent
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if i := strings.IndexByte(text, '#'); i >= 0 {
			text = text[:i]
		}
		fs := strings.Fields(text)
		if len(fs) == 0 {
			continue
		}
		if len(fs) < 2 {
			return fmt.Errorf("%s:%d: ожидается «время команда»", path, line)
		}
		at, err := strconv.ParseFloat(fs[0], 64)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, line, err)
		}
		args := fs[2:]
		var do func()
		switch fs[1] {
		case "key":
			if len(args) != 1 {
				return fmt.Errorf("%s:%d: key <имя>", path, line)
			}
			k, ok := host.KeyByName[args[0]]
			if !ok {
				if len(args[0]) != 1 {
					return fmt.Errorf("%s:%d: неизвестная клавиша %q", path, line, args[0])
				}
				k = host.CharKey(args[0][0])
			}
			do = func() { kb.Press(k.Scan, k.ASCII) }
		case "shot":
			if len(args) < 1 {
				return fmt.Errorf("%s:%d: shot <файл> [страница]", path, line)
			}
			name, page := args[0], 0
			if len(args) > 1 {
				page, _ = strconv.Atoi(args[1])
			}
			do = func() {
				out, err := os.Create(filepath.Join(outDir, name))
				if err != nil {
					panic(err)
				}
				defer out.Close()
				if err := png.Encode(out, scr.Snapshot(page)); err != nil {
					panic(err)
				}
			}
		case "quit":
			do = func() { panic(errQuit) }
		case "tickkey":
			if len(args) != 2 {
				return fmt.Errorf("%s:%d: tickkey <тик> <клавиша>", path, line)
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("%s:%d: %w", path, line, err)
			}
			if dbg.tickKeys == nil {
				dbg.tickKeys = map[int]string{}
			}
			dbg.tickKeys[n] = args[1]
			continue
		default:
			return fmt.Errorf("%s:%d: неизвестная команда %q", path, line, fs[1])
		}
		events = append(events, host.ScriptEvent{At: at, Do: do})
	}
	if err := sc.Err(); err != nil {
		return err
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].At < events[j].At })
	events = append(events, host.ScriptEvent{At: 3600, Do: func() { panic(errQuit) }})

	clk := host.NewScriptClock(events)
	g := game.New(scr, clk, kb, mixer, data, nil, scores, 12*3600)
	g.SetSpeed(speed)
	var onTick func(string)
	if dbg.stateLog != "" {
		lf, err := os.Create(dbg.stateLog)
		if err != nil {
			return err
		}
		defer lf.Close()
		lw := bufio.NewWriter(lf)
		defer lw.Flush()
		onTick = func(s string) { fmt.Fprintf(lw, "S %s\n", s) }
	}
	if dbg.tickKeys != nil {
		ticks := 0
		log := onTick
		onTick = func(s string) {
			if log != nil {
				log(s)
			}
			if name, ok := dbg.tickKeys[ticks]; ok {
				k := host.KeyByName[name]
				kb.Press(k.Scan, k.ASCII)
			}
			ticks++
		}
	}
	g.Debug(dbg.fixedRnd, dbg.startLevel, onTick)
	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok && errors.Is(e, errQuit) {
					return
				}
				err = fmt.Errorf("сбой игры: %v", r)
			}
		}()
		g.Run()
		return nil
	}()
	if err != nil {
		return err
	}
	if wavPath != "" {
		buf := make([]float32, int((clk.Now()+0.5)*sfx.Rate))
		mixer.Render(buf)
		return writeWAV(wavPath, buf)
	}
	return nil
}

func writeWAV(path string, pcm []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	n := uint32(len(pcm) * 2)
	hdr := []any{
		[4]byte{'R', 'I', 'F', 'F'}, 36 + n, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(1), uint32(sfx.Rate), uint32(sfx.Rate * 2), uint16(2), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, n,
	}
	for _, v := range hdr {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	for _, v := range pcm {
		s := int16(math.Max(-1, math.Min(1, float64(v))) * 32767)
		if err := binary.Write(w, binary.LittleEndian, s); err != nil {
			return err
		}
	}
	return w.Flush()
}
