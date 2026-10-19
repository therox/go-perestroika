package game

import (
	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/gfx"
)

// Run — main() оригинала (04e5:0008): заставка, затем цикл игр.
// Возвращается, когда игрок отказался «перестраиваться» дальше.
func (g *Game) Run() {
	g.scr.SetTextJustify(gfx.CenterText, gfx.CenterText)
	// титульная картинка и звуковая заставка
	g.scr.SetActivePage(0)
	g.scr.PutPixels(g.data.Title)
	g.work(0)
	g.introSample()
	g.flushKeys()

	g.scr.ClearDevice()
	g.setGamePalette()
	g.drawMainScreen()

	for g.playGame() {
	}
}

// introSample проигрывает звуковую заставку. В отличие от оригинала её можно
// прервать любой клавишей.
func (g *Game) introSample() {
	g.work(0)
	start := g.clk.Now()
	g.snd.Sync(start)
	d := g.snd.Sample(start, g.data.Intro, dat.SampleRate)
	for g.clk.Now()-start < d {
		if g.in.KeyHit() {
			g.snd.StopSamples(g.clk.Now())
			break
		}
		g.clk.Advance(0.02)
	}
}

// playGame — одна игра от первого этапа до «Примите соболезнования».
// Возвращает true, если игрок хочет сыграть ещё.
func (g *Game) playGame() bool {
	g.goodsGiven, g.goodsTaken = 0, 0
	g.curGiven, g.curTaken = 0, 0
	g.advGiven, g.advTaken = 0, 0
	g.advOutcome = [5]int{}
	g.cols = 5
	g.lives = 3
	g.lifeStep = 10000
	g.level = 0
	if g.startLevel > 1 {
		g.level = g.startLevel - 1
		g.cols = min(5+g.level, 12)
	}
	g.score = 0
	g.nextLife = 0
	g.startTime = g.now()

	for {
		g.burFrame = [2]int{0, 1}
		g.pose = 0
		g.cur, g.prev = 0, 0
		g.levelOK = true
		g.dead = false
		g.burActive[0] = g.rnd100()%4 != 0 && int32(g.level)*1000 < g.score
		g.cols++
		if g.cols > 13 {
			g.cols = 13
		}
		g.level++
		if g.level > 25 {
			return g.gameOver()
		}
		g.burActive[1] = g.level >= 15
		g.npads = g.genLevel()
		g.goal = g.npads - 1
		switch g.rnd100() % 3 {
		case 0:
			g.bur2Home = g.npads - 1 - g.cols
		case 1:
			g.bur2Home = g.goal
		default:
			g.bur2Home = g.cols
		}
		if g.level < 5 || g.rnd100()%3 != 0 {
			g.adventure = g.npads
		} else {
			g.adventure = g.rnd100()%(g.npads-10) + 8
			g.advGiven++
		}
		for g.currency[g.adventure] || g.goods[g.adventure] {
			g.adventure++
		}
		g.initPads()
		g.drawLevel()
		g.demX, g.oldX = g.padX[0], g.padX[0]
		g.demY, g.oldY = g.padY[0], g.padY[0]
		g.flushKeys()
		g.in.SetISR(true)

		for {
			if g.dead {
				g.levelOK = false
				g.lives--
				break
			}
			if g.burMove(0) && g.cur != g.goal {
				g.caught(0)
				g.levelOK = false
				g.lives--
				break
			}
			if g.burMove(1) && g.cur != g.goal {
				g.caught(1)
				g.levelOK = false
				g.lives--
				break
			}
			g.tick()
			g.sparkle(24, 30)
			if g.cur == g.goal {
				break
			}
		}
		if g.dead && g.levelOK {
			g.levelOK = false
			g.lives--
		}
		if g.levelOK {
			// этап пройден: Демократ на монете и фанфары
			g.burPad[0] = g.goal
			g.burErase(0)
			if g.oldX != g.demX || g.oldY != g.demY {
				g.eraseDemocrat(g.oldX, g.oldY)
			}
			g.fillCircle(g.padX[g.goal], g.padY[g.goal], g.r, colPadFill, colPadBorder)
			g.drawDemocrat(g.padX[g.goal], g.padY[g.goal])
			for _, f := range [3]int{500, 800, 1300} {
				g.sound(f)
				g.delay(100)
			}
			g.nosound()
		}
		g.in.SetISR(false)
		switch {
		case g.levelOK:
			g.score += int32(g.cols * 100)
		case g.lives != 0:
			// этап переигрывается
			g.cols--
			g.level--
		default:
			return g.gameOver()
		}
	}
}
