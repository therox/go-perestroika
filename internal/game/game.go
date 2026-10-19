// Package game — перенос логики игры «Перестройка» (Н. Скрипкин, 1990).
//
// Код повторяет структуру оригинальной программы на Turbo C: функции
// названы по смыслу, в комментариях указаны адреса исходных процедур
// в PERESTR.EXE (сегмент:смещение относительно образа).
//
// Игра выполняется последовательно в отдельной горутине, как и оригинальная
// программа. Время в ней течёт за счёт «стоимости» операций: каждая графическая
// операция и шаг логики тратят откалиброванное по оригиналу число инструкций,
// которое часы переводят в реальные паузы. Так сохраняется исходный темп,
// включая замедление на крупных уровнях.
package game

import (
	"math"
	"sync"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/gfx"
)

// Clock — время игры.
type Clock interface {
	// Now возвращает время игры в секундах.
	Now() float64
	// Advance продвигает время игры на sec секунд (с реальным ожиданием, если нужно).
	Advance(sec float64)
}

// Input — клавиатура. Пока установлен «обработчик прерывания» (SetISR(true)),
// нажатия попадают в очередь скан-кодов, иначе — в буфер BIOS.
type Input interface {
	SetISR(on bool)
	// KeyHit — аналог bioskey(1).
	KeyHit() bool
	// ReadKey — аналог bioskey(0) без ожидания: (скан<<8)|ASCII.
	ReadKey() (int, bool)
	// PopScan извлекает скан-код нажатия для обработчика прерывания.
	PopScan() (byte, bool)
	// Flush очищает буферы.
	Flush()
}

// Sound — звук: спикер и оцифрованные сэмплы.
type Sound interface {
	// Sync сообщает текущее время игры, чтобы звук не расходился с картинкой.
	Sync(t float64)
	Tone(t, freq float64)
	Sample(t float64, pwm []byte, rate float64) float64
	StopSamples(t float64)
}

// Storage сохраняет таблицу рекордов.
type Storage interface {
	SaveHiScores(hs [10]dat.HiScore) error
}

// Скорость эталонной машины в «инструкциях» в секунду (порядка AT-286).
const baseIPS = 2.5e6

const maxPads = 180

// Цвета (индексы палитры), заданные в init_graphics (04e5:04de).
const (
	colWater       = 1  // g_212e: узор воды
	colPadFill     = 2  // g_2126
	colPadBorder   = 11 // g_2112
	colStatusBar   = 3  // g_2166
	colStatusText  = 0  // g_213a
	colGoodsFill   = 9  // g_2162: «Товары народного потребления»
	colGoodsBorder = 0  // g_2172
	colCurFill     = 8  // g_214c: «Валютные операции»
	colCurBorder   = 0  // g_216e
	colTaxFill     = 7  // g_2144: «Прогрессивный налог»
	colTaxBorder   = 0  // g_216c
	colAdvFill     = 4  // g_2152: «Авантюра»
	colAdvBorder   = 0  // g_2170

	democratHalf = 13 // g_214e
	ballRadius   = 6  // g_213c
	burHalfW     = 24 // g_0096
	burHalfH     = 20 // g_0098
)

// Типы шариков-бонусов (аргумент F_0cee).
const (
	ballGoods = iota
	ballCurrency
	ballTax
	ballAdventure
)

// waterPattern — узор заливки воды (DGROUP:009e).
var waterPattern = [8]byte{0xf3, 0xcc, 0x3f, 0xff, 0xf3, 0xcc, 0x3f, 0xff}

// Game — состояние игры (глобальные переменные оригинала).
type Game struct {
	scr   *gfx.Screen
	clk   Clock
	in    Input
	snd   Sound
	data  *dat.Data
	store Storage

	speedMu sync.Mutex
	speed   float64

	rng       uint32
	timeShift float64
	aspect    float64

	// Геометрия уровня.
	cols    int // g_1c18: число столбцов-1
	r       int // g_2132: полный радиус листа
	x0      int // g_1fae: x левого столбца
	xMax    int // g_1fd0: x правого столбца
	yTop    int // g_2036: y верхнего ряда
	yBot    int // g_1c20: y нижнего ряда
	colStep int // g_2120
	rowStep int // g_2128
	npads   int // g_2154

	padX, padY [maxPads]int   // g_1238, g_13a0
	padSize    [maxPads]int   // g_1184: текущий радиус листа
	counter    [maxPads]int32 // g_1cde: >0 — на плаву, <0 — под водой
	floatT     [maxPads]int32 // g_1946
	sinkT      [maxPads]int32 // g_150c
	goods      [maxPads]bool  // g_1c26
	currency   [maxPads]bool  // g_1890
	tax        [maxPads]bool  // g_17dc

	adventure int // g_2114: лист с «Авантюрой» (npads — нет)
	goal      int // g_2080: лист с монетой этапа

	cur, prev  int // g_2034, g_2086
	demX, demY int // g_2094, g_2096: куда прыгает Демократ
	oldX, oldY int // g_211c, g_2122: где он нарисован
	pose       int // g_2124

	burActive   [2]bool // g_009a, g_009c
	burPad      [2]int  // g_2078, g_1508
	burFrame    [2]int  // g_2134, g_1944
	bur2Home    int     // g_150a
	burThrottle int     // g_00a8

	level    int   // g_208e
	lives    int   // g_1fd2
	score    int32 // g_2098
	nextLife int32 // g_215e
	lifeStep int32 // g_2168

	pending bool // g_0094: ход нажат, но ещё не выполнен
	dead    bool // g_2146
	levelOK bool // g_2140
	pause   bool // g_00a6

	goodsGiven, goodsTaken int    // g_2148, g_211a
	curGiven, curTaken     int    // g_214a, g_211e
	advGiven, advTaken     int    // g_2150, g_2136
	advOutcome             [5]int // g_2104..g_210c
	startTime, endTime     float64

	hiScores [10]dat.HiScore

	// отладка и сверка с оригиналом
	fixedRnd   int                // если >0 — F_0fb4 всегда возвращает это значение
	startLevel int                // если >0 — игра начинается с этого этапа
	onTick     func(state string) // вызывается в начале каждого тика
}

// Debug задаёт отладочные параметры: фиксированное значение «часового»
// генератора F_0fb4 (делает игру детерминированной), начальный этап и
// функцию, получающую состояние игры в начале каждого тика.
func (g *Game) Debug(fixedRnd, startLevel int, onTick func(string)) {
	g.fixedRnd = fixedRnd
	g.startLevel = startLevel
	g.onTick = onTick
}

// New создаёт игру.
func New(scr *gfx.Screen, clk Clock, in Input, snd Sound, data *dat.Data, store Storage, hs [10]dat.HiScore, timeShift float64) *Game {
	return &Game{
		scr: scr, clk: clk, in: in, snd: snd, data: data, store: store,
		speed: 1, rng: 1, timeShift: timeShift, hiScores: hs,
		// getaspectratio() для EGAHI: 7750/10000
		aspect: 0.775,
	}
}

// SetSpeed задаёт множитель скорости относительно эталонной машины.
func (g *Game) SetSpeed(v float64) {
	g.speedMu.Lock()
	g.speed = math.Max(0.1, math.Min(v, 20))
	g.speedMu.Unlock()
}

// Speed возвращает множитель скорости.
func (g *Game) Speed() float64 {
	g.speedMu.Lock()
	defer g.speedMu.Unlock()
	return g.speed
}

// work учитывает стоимость логики (в инструкциях) и продвигает время на
// накопленную стоимость графики и логики.
func (g *Game) work(units float64) {
	total := units + g.scr.TakeCost()
	if total > 0 {
		g.clk.Advance(total / (baseIPS * g.Speed()))
	}
}

// delay — аналог delay(ms): реальная задержка, не зависящая от скорости машины.
func (g *Game) delay(ms int) {
	g.work(0)
	g.clk.Advance(float64(ms) / 1000)
}

// rand — генератор Turbo C 2.0 (srand в оригинале не вызывался).
func (g *Game) rand() int {
	g.rng = g.rng*0x015A4E35 + 1
	return int((g.rng >> 16) & 0x7FFF)
}

// rnd100 — F_0fb4: сотые доли секунды системных часов DOS + 1.
// Часы DOS шли от таймера 18.2 Гц, поэтому значения квантованы.
func (g *Game) rnd100() int {
	if g.fixedRnd > 0 {
		return g.fixedRnd
	}
	t := g.clk.Now() + g.timeShift
	ticks := math.Floor(t * 1193180 / 65536)
	hund := int64(ticks*65536/1193180*100) % 100
	return int(hund) + 1
}

// now — F_4d2b: текущее время суток в секундах.
func (g *Game) now() float64 {
	return g.clk.Now() + g.timeShift
}

// sound/nosound — спикер. Как и sound() в Turbo C, частоты до 18 Гц
// игнорируются: продолжает звучать прежний тон.
func (g *Game) sound(freq int) {
	g.work(0)
	g.snd.Sync(g.clk.Now())
	if freq > 18 {
		g.snd.Tone(g.clk.Now(), float64(freq))
	}
}

func (g *Game) nosound() {
	g.work(0)
	g.snd.Sync(g.clk.Now())
	g.snd.Tone(g.clk.Now(), 0)
}

// sweep воспроизводит цикл вида for(f=from; f!=to; f+=step) sound(f*mul):
// частота меняется на каждой итерации, длительность итерации — как у оригинала.
func (g *Game) sweep(from, to, step int, mul func(f int) int) {
	g.work(0)
	g.snd.Sync(g.clk.Now())
	const iterCost = 29
	dt := iterCost / (baseIPS * g.Speed())
	t := g.clk.Now()
	n := 0
	for f := from; (step > 0 && f < to) || (step < 0 && f > to); f += step {
		v := f
		if mul != nil {
			v = mul(f)
		}
		if v > 18 {
			g.snd.Tone(t+float64(n)*dt, float64(v))
		}
		n++
	}
	g.clk.Advance(float64(n) * dt)
}

// beep2 — частый сигнал оригинала: sound(800) delay(20) sound(200) delay(40) nosound().
func (g *Game) beep2() {
	g.sound(800)
	g.delay(20)
	g.sound(200)
	g.delay(40)
	g.nosound()
}

// playSample — playsample(): блокирующее воспроизведение оцифрованного звука.
func (g *Game) playSample(pwm []byte) {
	g.work(0)
	g.snd.Sync(g.clk.Now())
	g.snd.Tone(g.clk.Now(), 0)
	d := g.snd.Sample(g.clk.Now(), pwm, dat.SampleRate)
	g.clk.Advance(d)
}

// waitKey — getch(): ожидание любой клавиши.
func (g *Game) waitKey() int {
	g.work(0)
	for {
		if k, ok := g.in.ReadKey(); ok {
			return k
		}
		g.clk.Advance(0.01)
	}
}

// flushKeys — while (bioskey(1)) bioskey(0);
func (g *Game) flushKeys() {
	for g.in.KeyHit() {
		g.in.ReadKey()
	}
}

// pollKey — F_4cd8: клавиша без ожидания; расширенные коды как 0x100+скан.
// bioskey() в Turbo C при каждом вызове трижды обращался к DOS (функции
// 25h/35h — перехват Ctrl-Break) и один раз к BIOS: около 400 инструкций.
func (g *Game) pollKey() int {
	g.work(400)
	k, ok := g.in.ReadKey()
	if !ok {
		return 0
	}
	if k&0xFF == 0 {
		return k>>8 + 0x100
	}
	return k & 0xFF
}

// getKey — F_0fd1: клавиша с ожиданием; расширенные коды как 0x100+скан.
func (g *Game) getKey() int {
	k := g.waitKey()
	if k&0xFF == 0 {
		return k>>8 + 0x100
	}
	return k & 0xFF
}

// ftol — приведение double к int с отбрасыванием дробной части, как __ftol.
func ftol(v float64) int {
	return int(math.Trunc(v))
}
