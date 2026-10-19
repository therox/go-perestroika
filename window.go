package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/game"
	"github.com/therox/go-perestroika/internal/gfx"
	"github.com/therox/go-perestroika/internal/host"
	"github.com/therox/go-perestroika/internal/sfx"
)

const windowTitle = "Перестройка"

// Задержка автоповтора клавиш PC: 0,5 с, затем ~11 нажатий в секунду (в тиках по 1/60 с).
const (
	repeatDelay = 30
	repeatEvery = 5
)

// specialKeys — служебные клавиши окна и их коды PC.
var specialKeys = map[ebiten.Key]string{
	ebiten.KeyEscape:      "esc",
	ebiten.KeyEnter:       "enter",
	ebiten.KeyNumpadEnter: "enter",
	ebiten.KeyBackspace:   "bksp",
	ebiten.KeyTab:         "tab",
	ebiten.KeyArrowUp:     "up",
	ebiten.KeyArrowDown:   "down",
	ebiten.KeyArrowLeft:   "left",
	ebiten.KeyArrowRight:  "right",
	ebiten.KeyHome:        "home",
	ebiten.KeyEnd:         "end",
	ebiten.KeyPageUp:      "pgup",
	ebiten.KeyPageDown:    "pgdn",
	ebiten.KeyInsert:      "ins",
	ebiten.KeyDelete:      "del",
	ebiten.KeyF1:          "f1",
	ebiten.KeyF2:          "f2",
	ebiten.KeyF3:          "f3",
	ebiten.KeyF4:          "f4",
}

// Стрелки дополнительной клавиатуры (при выключенном NumLock на PC
// они давали те же скан-коды, что и стрелки).
var numpadArrows = map[ebiten.Key]string{
	ebiten.KeyNumpad8: "up",
	ebiten.KeyNumpad2: "down",
	ebiten.KeyNumpad4: "left",
	ebiten.KeyNumpad6: "right",
}

type app struct {
	scr   *gfx.Screen
	kb    *host.Keyboard
	g     *game.Game
	done  chan struct{}
	mixer *sfx.Mixer
	store *host.HiScoreFile

	frame *ebiten.Image
	pre   *ebiten.Image
	preK  [2]int
	pix   []byte
	keys  []ebiten.Key
	runes []rune
	speed float64

	// показ кадров с задержкой, чтобы картинка совпадала с запаздывающим
	// звуком (Bluetooth-наушники и колонки)
	settings host.Settings
	history  []timedFrame
	spare    []*gfx.Frame

	// отладка: сохранить кадр окна через captureAt кадров и выйти
	capturePath string
	captureAt   int
	frames      int
}

// timedFrame — снимок экрана и момент, когда он был снят.
type timedFrame struct {
	at    time.Time
	frame *gfx.Frame
}

// Пределы и шаг компенсации задержки звука.
const (
	maxAudioDelayMs  = 800
	audioDelayStepMs = 20
)

func runWindow(data *dat.Data, dataErr error, store *host.HiScoreFile, speed, volume float64, fullscreen bool, capture string, audioDelayMs int) error {
	const audioBuffer = 25 * time.Millisecond
	mixer := sfx.NewMixer(0.005, audioBuffer.Seconds())
	mixer.SetVolume(volume)
	actx := audio.NewContext(sfx.Rate)
	player, err := actx.NewPlayerF32(mixer)
	if err != nil {
		return err
	}
	player.SetBufferSize(audioBuffer)
	player.Play()

	a := &app{
		scr:   gfx.NewScreen(),
		kb:    &host.Keyboard{},
		done:  make(chan struct{}),
		mixer: mixer,
		store: store,
		frame: ebiten.NewImage(gfx.W, gfx.H),
		pix:   make([]byte, gfx.W*gfx.H*4),
		speed: speed,

		settings: host.LoadSettings(),

		capturePath: capture,
		captureAt:   180,
	}
	if audioDelayMs >= 0 {
		a.settings.AudioDelayMs = audioDelayMs
	}
	a.settings.AudioDelayMs = min(max(a.settings.AudioDelayMs, 0), maxAudioDelayMs)
	if data != nil {
		a.start(data)
	} else {
		drawNoData(a.scr, dataErr)
	}

	ebiten.SetWindowTitle(a.title())
	ebiten.SetWindowSize(960, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetScreenClearedEveryFrame(true)
	if capture != "" {
		// невидимому окну композитор может не отдавать кадры; для снимка vsync не нужен
		ebiten.SetVsyncEnabled(false)
	}
	return ebiten.RunGame(a)
}

// start запускает игру, когда файл данных есть.
func (a *app) start(data *dat.Data) {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	a.g = game.New(a.scr, host.NewRealClock(), a.kb, a.mixer, data, a.store,
		a.store.Load(data.HiScores), now.Sub(dayStart).Seconds())
	a.g.SetSpeed(a.speed)
	ebiten.SetWindowIcon(windowIcons(data))
	go func() {
		defer close(a.done)
		a.g.Run()
	}()
}

// waitData обрабатывает перетаскивание файла данных в окно.
func (a *app) waitData() {
	dropped := ebiten.DroppedFiles()
	if dropped == nil {
		return
	}
	raw, err := findDropped(dropped)
	if err == nil {
		var d *dat.Data
		if d, err = host.InstallDataFile(raw); err == nil {
			a.start(d)
			return
		}
	}
	drawNoData(a.scr, err)
}

// findDropped ищет TOPPLER.DAT среди перетащенных файлов и папок.
func findDropped(fsys fs.FS) ([]byte, error) {
	var found string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), host.DataFileName) {
			found = p
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if found == "" {
		return nil, host.ErrNoData
	}
	return fs.ReadFile(fsys, found)
}

// drawNoData выводит в окне подсказку, где взять файл данных.
func drawNoData(scr *gfx.Screen, err error) {
	scr.SetActivePage(0)
	scr.ResetPalette()
	scr.SetFillStyle(gfx.SolidFill, 1)
	scr.Bar(0, 0, gfx.MaxX, gfx.MaxY)
	scr.SetFillStyle(gfx.SolidFill, 7)
	scr.SetColor(0)
	scr.Bar(40, 100, 599, 249)
	scr.Rectangle(40, 100, 599, 249)
	scr.SetTextJustify(gfx.CenterText, gfx.CenterText)
	line := func(y, color int, text string) {
		scr.SetColor(color)
		scr.OutTextXY(gfx.W/2, y, cp866(text))
	}
	switch {
	case err == nil || errors.Is(err, host.ErrNoData):
		line(125, 4, "Не найден файл данных "+host.DataFileName)
	case errors.Is(err, dat.ErrFormat):
		line(125, 4, "Этот файл "+host.DataFileName+" не подходит")
	default:
		line(125, 4, "Не удалось прочитать файл данных")
		msg := []rune(err.Error())
		if len(msg) > 70 {
			msg = append([]rune("..."), msg[len(msg)-67:]...)
		}
		line(225, 4, string(msg))
	}
	line(155, 0, "Для игры нужен файл "+host.DataFileName+" из \"Перестройки\"")
	line(170, 0, "1990 года (версия 2.2). Перетащите его в это окно")
	line(185, 0, "или положите рядом с программой.")
	scr.TakeCost()
}

// cp866 переводит строку в кодировку шрифта 8x8.
func cp866(s string) []byte {
	var b []byte
	for _, r := range s {
		if c, ok := dat.RuneToCP866(r); ok {
			b = append(b, c)
		} else {
			b = append(b, '?')
		}
	}
	return b
}

func (a *app) title() string {
	t := windowTitle
	if math.Abs(a.speed-1) > 1e-9 {
		t += fmt.Sprintf(" — скорость ×%.2g", a.speed)
	}
	if a.settings.AudioDelayMs > 0 {
		t += fmt.Sprintf(" — задержка звука %d мс", a.settings.AudioDelayMs)
	}
	return t
}

func (a *app) press(name string) {
	k := host.KeyByName[name]
	a.kb.Press(k.Scan, k.ASCII)
}

func (a *app) Update() error {
	select {
	case <-a.done:
		return ebiten.Termination
	default:
	}
	if a.capturePath != "" && a.frames > a.captureAt {
		return ebiten.Termination
	}
	if a.g == nil {
		a.waitData()
	}

	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (alt && inpututil.IsKeyJustPressed(ebiten.KeyEnter)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) || inpututil.IsKeyJustPressed(ebiten.KeyF8) || inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyF7):
			a.speed = math.Max(0.25, a.speed/1.25)
		case inpututil.IsKeyJustPressed(ebiten.KeyF8):
			a.speed = math.Min(8, a.speed*1.25)
		default:
			a.speed = 1
		}
		if a.g != nil {
			a.g.SetSpeed(a.speed)
		}
		ebiten.SetWindowTitle(a.title())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) || inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		step := audioDelayStepMs
		if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
			step = -step
		}
		a.settings.AudioDelayMs = min(max(a.settings.AudioDelayMs+step, 0), maxAudioDelayMs)
		_ = host.SaveSettings(a.settings)
		ebiten.SetWindowTitle(a.title())
	}

	// служебные клавиши с автоповтором
	for k, name := range specialKeys {
		if k == ebiten.KeyEnter && alt {
			continue
		}
		if repeatPressed(k) {
			a.press(name)
		}
	}
	// стрелки на цифровом блоке
	for k, name := range numpadArrows {
		if repeatPressed(k) {
			a.press(name)
		}
	}
	// печатные символы — с учётом раскладки клавиатуры
	a.runes = ebiten.AppendInputChars(a.runes[:0])
	for _, r := range a.runes {
		if c := host.RuneToCP866(r); c != 0 {
			k := host.CharKey(c)
			a.kb.Press(k.Scan, k.ASCII)
		}
	}
	return nil
}

// repeatPressed — нажатие клавиши с автоповтором, как у клавиатуры PC.
func repeatPressed(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > repeatDelay && (d-repeatDelay)%repeatEvery == 0)
}

// delayedFrame снимает текущий кадр и возвращает тот, что был на экране
// AudioDelayMs миллисекунд назад.
func (a *app) delayedFrame() *gfx.Frame {
	now := time.Now()
	var f *gfx.Frame
	if n := len(a.spare); n > 0 {
		f, a.spare = a.spare[n-1], a.spare[:n-1]
	} else {
		f = gfx.NewFrame()
	}
	a.scr.Capture(f)
	a.history = append(a.history, timedFrame{now, f})

	cutoff := now.Add(-time.Duration(a.settings.AudioDelayMs) * time.Millisecond)
	// самый свежий кадр не позже cutoff; более старые больше не понадобятся
	k := 0
	for k+1 < len(a.history) && !a.history[k+1].at.After(cutoff) {
		k++
	}
	for _, old := range a.history[:k] {
		a.spare = append(a.spare, old.frame)
	}
	a.history = append(a.history[:0], a.history[k:]...)
	return a.history[0].frame
}

func (a *app) Draw(screen *ebiten.Image) {
	a.delayedFrame().RGBA(a.pix)
	a.frame.WritePixels(a.pix)

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	screen.Fill(color.Black)
	// изображение EGA растягивается до пропорций 4:3, как на мониторе того времени
	w := math.Min(float64(sw), float64(sh)*4/3)
	h := w * 3 / 4
	// «чёткий» билинейный: целочисленное увеличение ближайшим соседом,
	// затем плавное масштабирование до нужного размера
	kx := max(1, int(math.Ceil(w/gfx.W)))
	ky := max(1, int(math.Ceil(h/gfx.H)))
	if a.pre == nil || a.preK != [2]int{kx, ky} {
		if a.pre != nil {
			a.pre.Deallocate()
		}
		a.pre = ebiten.NewImage(gfx.W*kx, gfx.H*ky)
		a.preK = [2]int{kx, ky}
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(kx), float64(ky))
	a.pre.DrawImage(a.frame, &op)

	op = ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(w/float64(gfx.W*kx), h/float64(gfx.H*ky))
	op.GeoM.Translate((float64(sw)-w)/2, (float64(sh)-h)/2)
	screen.DrawImage(a.pre, &op)

	a.frames++
	if a.capturePath != "" && a.frames == a.captureAt {
		saveScreen(screen, a.capturePath)
	}
}

// saveScreen сохраняет кадр окна в PNG (для отладки отрисовки).
func saveScreen(screen *ebiten.Image, path string) {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}

func (a *app) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}

// windowIcons строит значки окна из спрайта Демократа.
func windowIcons(data *dat.Data) []image.Image {
	spr := data.Democrat[2]
	pal := game.Palette()
	side := max(spr.W, spr.H)
	var icons []image.Image
	for _, size := range []int{16, 32, 48, 64} {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				sx := x*side/size - (side-spr.W)/2
				sy := y*side/size - (side-spr.H)/2
				if sx < 0 || sy < 0 || sx >= spr.W || sy >= spr.H {
					continue
				}
				if c, ok := spr.At(sx, sy); ok {
					img.Set(x, y, gfx.EGAColor(pal[c]))
				}
			}
		}
		icons = append(icons, img)
	}
	return icons
}
