package game

import "github.com/therox/go-perestroika/internal/gfx"

// Скан-коды клавиш, которые понимал обработчик прерывания клавиатуры (04e5:2420).
const (
	scanEsc   = 0x01
	scanUp    = 0x48
	scanLeft  = 0x4B
	scanRight = 0x4D
	scanDown  = 0x50
)

// pollISR выполняет логику обработчика прерывания клавиатуры (04e5:2420)
// для одного накопившегося нажатия. В оригинале обработчик срабатывал
// асинхронно; здесь он вызывается в тех местах, где игра проверяла его флаги.
func (g *Game) pollISR() {
	sc, ok := g.in.PopScan()
	if !ok || g.pending || g.dead {
		return
	}
	switch sc {
	case scanUp:
		if g.yTop+g.rowStep <= g.demY {
			g.prev = g.cur
			g.pose = 0
			g.cur += g.cols + 1
			g.demY -= g.rowStep
			g.pending = true
		}
	case scanLeft:
		if g.x0+g.colStep <= g.demX {
			g.prev = g.cur
			g.pose = 3
			g.cur--
			g.demX -= g.colStep
			g.pending = true
		}
	case scanRight:
		if g.xMax-g.colStep >= g.demX {
			g.prev = g.cur
			g.pose = 1
			g.cur++
			g.demX += g.colStep
			g.pending = true
		}
	case scanDown:
		if g.yBot-g.rowStep >= g.demY {
			g.prev = g.cur
			g.pose = 2
			g.cur -= g.cols + 1
			g.demY += g.rowStep
			g.pending = true
		}
	case scanEsc:
		g.pause = true
	}
}

// tick — F_14a2: один проход по всем листьям.
func (g *Game) tick() {
	if g.onTick != nil {
		g.onTick(g.stateLine())
	}
	for i := 0; i < g.npads; i++ {
		g.pollISR()
		g.work(120)
		if g.pause {
			g.doPause()
		}
		if g.burActive[0] && g.cur == g.burPad[0] {
			g.caught(0)
			g.dead = true
			break
		}
		if g.burActive[1] && g.cur == g.burPad[1] {
			g.caught(1)
			g.dead = true
			break
		}
		// неподвижные листья: «налог», «авантюра», цель и «дом» второго Бюрократа
		if g.tax[i] || i == g.adventure || i == g.goal {
			continue
		}
		if i == g.bur2Home && g.burActive[1] && g.burPad[1] == g.bur2Home {
			continue
		}
		x, y := g.padX[i], g.padY[i]
		switch c := g.counter[i]; {
		case c > 1:
			g.counter[i]--
			ns := int(int32(g.r-8)*g.counter[i]/g.floatT[i]) + 10
			if g.padSize[i] > ns {
				g.eraseCell(x, y)
				g.fillCircle(x, y, ns, colPadFill, colPadBorder)
				g.padSize[i] = ns
				switch {
				case x == g.demX && y == g.demY:
					g.drawDemocrat(g.demX, g.demY)
				case g.goods[i]:
					g.drawBall(x, y, ballGoods)
				case g.tax[i]:
					g.drawBall(x, y, ballTax)
				case g.currency[i]:
					g.drawBall(x, y, ballCurrency)
				}
				for w := 0; w < 2; w++ {
					if g.burPad[w] == i && g.burActive[w] {
						g.drawBureaucrat(w, false)
						g.nextFrame(w)
					}
				}
			}
		case c == 1:
			// лист тонет
			g.counter[i] = g.sinkT[i]
			if g.cur == i {
				g.pending = true
				g.dead = true
				g.eraseCell(x, y)
				g.scr.SetActivePage(0)
				g.eraseCell(x, y)
				g.scr.SetActivePage(1)
				g.drown()
				g.endTick()
				return
			}
			switch {
			case g.burPad[0] == i && g.burActive[0]:
				g.burDrown(i, 0)
			case g.burPad[1] == i && g.burActive[1]:
				g.burDrown(i, 1)
			default:
				g.eraseCell(x, y)
				g.ripples(x, y)
			}
		case c == -1:
			// лист всплывает
			g.counter[i] = g.floatT[i]
			g.padSize[i] = g.r
			g.fillCircle(x, y, g.r, colPadFill, colPadBorder)
			if g.goods[i] {
				g.drawBall(x, y, ballGoods)
			}
			if g.currency[i] {
				g.drawBall(x, y, ballCurrency)
			}
		case c < -1:
			g.counter[i]++
		}
		if g.pending && !g.dead {
			g.commitMove()
			g.land()
		}
		if g.level >= 10 {
			g.goalShift()
		}
		if g.dead || g.cur == g.goal {
			break
		}
	}
	g.endTick()
}

func (g *Game) endTick() {
	g.scr.CopyRect(0, 0, gfx.MaxX, gfx.MaxY, 1, 0, 0, 0)
	g.work(0)
}

// commitMove — F_1010: перерисовать Демократа на новом листе.
func (g *Game) commitMove() {
	x, y := g.demX, g.demY
	if g.cur == g.goal {
		g.fillCircle(g.padX[g.goal], g.padY[g.goal], g.r, colPadFill, colPadBorder)
	}
	if g.dead {
		return
	}
	onBur := func(p int) bool {
		return (g.burActive[0] && p == g.burPad[0]) || (g.burActive[1] && p == g.burPad[1])
	}
	if !onBur(g.prev) {
		g.eraseDemocrat(g.oldX, g.oldY)
	}
	g.sound(200)
	if !onBur(g.cur) {
		g.drawDemocrat(x, y)
	}
	g.oldX, g.demX = x, x
	g.oldY, g.demY = y, y
	g.work(0)
	g.nosound()
	g.pending = false
}

// land — F_1b3c: что случилось после прыжка на лист. Возвращает false, если Демократ утонул.
func (g *Game) land() bool {
	c := g.cur
	if g.counter[c] < 1 {
		g.dead = true
		g.drown()
		return false
	}
	switch {
	case g.goods[c]:
		g.goods[c] = false
		g.score += int32(g.level * 50)
		g.goodsTaken++
		g.playSample(g.data.Sounds[0])
		g.drawScore()
	case c == g.adventure:
		g.adventureEvent()
	case g.tax[c]:
		g.tax[c] = false
		if int32(g.level*250) < g.score {
			g.score -= int32(g.level * 250)
		} else {
			g.score = 0
		}
		g.playSample(g.data.Sounds[3])
		g.drawScore()
	case g.currency[c]:
		g.currency[c] = false
		g.score += int32(g.level * 1000)
		g.curTaken++
		g.playSample(g.data.Sounds[2])
		g.drawScore()
	}
	// дополнительная жизнь за каждые 10000, 20000, 40000... очков
	if g.score > g.nextLife && g.score-g.nextLife >= g.lifeStep {
		g.nextLife += g.lifeStep
		g.lifeStep *= 2
		if g.lives < 6 {
			g.extraLife()
		}
	}
	return true
}

// adventureEvent — «Авантюра»: случайное событие (часть F_1b3c).
func (g *Game) adventureEvent() {
	g.advTaken++
	g.adventure = g.npads
	switch g.rnd100() % 5 {
	case 0: // «Жизнь демократа»
		if g.lives < 6 {
			g.extraLife()
		}
		g.advOutcome[0]++
	case 1: // «Смерть демократа»
		g.drown()
		g.dead = true
		g.advOutcome[1]++
	case 2: // «Материальное поощрение»
		g.advOutcome[2]++
		g.score += int32(g.level) * 2000
		g.sweep(300, 16000, 1, nil)
		g.nosound()
		g.drawScore()
	case 3:
		g.adventureBureaucrat(1)
	case 4:
		g.adventureBureaucrat(0)
	}
}

// adventureBureaucrat — исходы «Авантюры» 3 и 4: Бюрократ исчезает или появляется.
func (g *Game) adventureBureaucrat(which int) {
	if g.burActive[which] {
		// «Смерть бюрократа»
		g.burErase(which)
		p := g.burPad[which]
		g.fillCircle(g.padX[p], g.padY[p], g.padSize[p], colPadFill, colPadBorder)
		switch {
		case p == g.goal:
			g.drawCoin(g.padX[p], g.padY[p])
		case g.goods[p]:
			g.drawBall(g.padX[p], g.padY[p], ballGoods)
		case g.tax[p]:
			g.drawBall(g.padX[p], g.padY[p], ballTax)
		case g.currency[p]:
			g.drawBall(g.padX[p], g.padY[p], ballCurrency)
		}
		g.whistle()
		if which == 1 {
			g.burPad[1] = g.bur2Home
		} else {
			g.burPad[0] = g.goal
		}
		g.burActive[which] = false
		g.advOutcome[3]++
		return
	}
	// «Жизнь бюрократа»
	if which == 1 {
		if g.burActive[0] && g.burPad[0] != g.goal {
			g.bur2Home = g.goal
		}
		g.burPad[1] = g.bur2Home
	} else {
		g.burPad[0] = g.goal
	}
	g.advOutcome[4]++
	g.beep2()
	g.burActive[which] = true
}

// extraLife — F_200e: прибавить жизнь.
func (g *Game) extraLife() {
	g.lives++
	g.drawLives()
	for _, f := range [3]int{200, 1200, 500} {
		g.sound(f)
		g.delay(25)
	}
	g.nosound()
}

// doPause — F_4dbd: пауза по ESC с показом таблицы рекордов.
func (g *Game) doPause() {
	g.in.SetISR(false)
	g.scr.SetActivePage(0)
	g.showHiScores()
	g.scr.SetActivePage(1)
	g.pause = false
	g.in.SetISR(true)
}

// drown — F_205c: Демократ тонет (круги на воде и крест).
func (g *Game) drown() {
	x, y := g.demX, g.demY
	if x != g.oldX || y != g.oldY {
		g.eraseDemocrat(g.oldX, g.oldY)
	}
	g.setWater()
	g.scr.SetActivePage(0)
	g.scr.SetColor(colPadBorder)
	g.scr.Bar(x-g.r, y-g.r-2, x+g.r, y+g.r-2)
	for k := 1; k < 4; k++ {
		g.scr.Ellipse(x, y, k*8, ftol(float64(k*8)*g.aspect*0.7))
		g.sweep(50, 2000, 1, nil)
		m := g.rand()%3 + 1
		g.sweep(500, 2000, 1, func(f int) int { return f * m })
		g.nosound()
		g.delay(30)
	}
	dy := 24 * g.aspect
	g.scr.Bar(x-24, ftol(float64(y)-dy), x+24, ftol(float64(y)+dy))
	g.drawSprite(x-20, y-22, 0, g.data.Cross, false)
	g.scr.SetActivePage(1)
	g.scr.Bar(x-g.r, y-g.r-2, x+g.r, y+g.r-2)
	g.delay(2000)
}

// whistle — F_2261: свист (исчезновение Бюрократа).
func (g *Game) whistle() {
	double := func(f int) int { return 2 * f }
	g.sweep(10000, 50, -1, double)
	g.sweep(10000, 50, -1, double)
	g.sweep(50, 10000, 1, double)
}

// ripples — F_22c0: круги на воде от утонувшего листа.
func (g *Game) ripples(x, y int) {
	g.scr.CopyRect(0, 0, gfx.MaxX, gfx.MaxY, 1, 0, 0, 0)
	g.scr.SetActivePage(0)
	g.scr.SetColor(colPadBorder)
	g.scr.Ellipse(x, y, 8, ftol(8*g.aspect*0.7))
	g.scr.Ellipse(x, y, 16, ftol(8*g.aspect*1.4))
	m := g.rand()%2 + 1
	g.sweep(1000, 3000, 1, func(f int) int { return f * m })
	g.pollISR()
	if g.pending {
		g.scr.SetActivePage(1)
		g.commitMove()
		g.land()
		g.scr.SetActivePage(0)
	}
	m = g.rand()%3 + 1
	g.sweep(3000, 5000, 1, func(f int) int { return f * m })
	g.nosound()
	g.scr.SetActivePage(1)
	g.pollISR()
	if g.pending {
		g.commitMove()
		g.land()
	}
}
