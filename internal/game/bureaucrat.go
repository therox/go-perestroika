package game

import "github.com/therox/go-perestroika/internal/gfx"

// burErase — F_3490: стереть Бюрократа which вместе с листом.
func (g *Game) burErase(which int) {
	p := g.burPad[which]
	x, y := g.padX[p], g.padY[p]
	g.setWater()
	if sz := g.padSize[p]; sz > burHalfW {
		g.scr.Bar(x-sz, y-sz, x+sz, y+sz)
	} else {
		g.scr.Bar(x-burHalfW, y-burHalfH, x+burHalfW, y+burHalfH)
	}
}

// burMove — F_357b: шаг Бюрократа which к Демократу. Возвращает true, если поймал.
func (g *Game) burMove(which int) bool {
	g.pollISR()
	moved := true
	if g.pending {
		g.commitMove()
		g.land()
	}
	// с 20-го этапа первый Бюрократ всегда возвращается
	if !g.burActive[0] && g.level >= 20 {
		g.burPad[0] = g.goal
		g.burActive[0] = true
	}
	pad := g.burPad[which]
	other := g.burPad[1-which]
	frame := g.burFrame[which]
	if !g.burActive[which] || g.dead {
		return false
	}
	g.burThrottle++
	if g.burThrottle < 1 {
		return false
	}
	g.burThrottle = 0
	g.work(300)

	old := pad
	g.burErase(which)
	g.fillCircle(g.padX[pad], g.padY[pad], g.padSize[pad], colPadFill, colPadBorder)
	switch {
	case pad == g.goal:
		g.drawCoin(g.padX[pad], g.padY[pad])
	case g.goods[pad]:
		// Бюрократ превращает «Товары» в «Прогрессивный налог»
		if g.rnd100() > g.level*3+25 {
			g.drawBall(g.padX[pad], g.padY[pad], ballGoods)
		} else {
			g.drawBall(g.padX[pad], g.padY[pad], ballTax)
			g.tax[pad] = true
			g.goods[pad] = false
		}
	case g.currency[pad]:
		g.drawBall(g.padX[pad], g.padY[pad], ballCurrency)
	case pad == g.adventure:
		g.drawBall(g.padX[pad], g.padY[pad], ballAdventure)
	case g.tax[pad]:
		g.drawBall(g.padX[pad], g.padY[pad], ballTax)
	}

	// сначала по горизонтали
	if g.padX[pad] < g.demX && pad+1 != other {
		pad++
	} else if g.padX[pad] > g.demX && pad-1 != other {
		pad--
	}
	ok := old != pad && g.counter[pad] > 2
	if !ok {
		pad = old
		// затем по вертикали
		step := g.cols + 1
		if g.padY[pad] < g.demY && pad-step != other {
			pad -= step
		} else if g.padY[pad] > g.demY && pad+step != other {
			pad += step
		} else {
			moved = false
		}
		if g.counter[pad] <= 2 {
			pad = old
			moved = false
			if g.counter[pad] < 2 && g.burEscape(which) {
				moved = true
				pad = g.burPad[which]
			}
		}
	}

	g.fillCircle(g.padX[pad], g.padY[pad], g.padSize[pad], colPadFill, colPadBorder)
	if pad == g.goal {
		g.drawCoin(g.padX[pad], g.padY[pad])
	}
	g.drawSprite(g.padX[pad]-burHalfW/2-6, g.padY[pad]-burHalfH/2-9, 1, g.data.Bureaucrat[frame], false)
	frame++
	if frame > 7 {
		frame = 0
	}
	g.scr.CopyRect(0, 0, gfx.MaxX, gfx.MaxY, 1, 0, 0, 0)
	g.work(0)
	if moved {
		g.beep2()
	}
	g.burPad[which] = pad
	g.burFrame[which] = frame
	g.pollISR()
	if g.pending {
		g.commitMove()
		g.land()
	}
	return g.cur == pad
}

// burEscape — F_39b5: Бюрократ перепрыгивает с тонущего листа на соседний,
// который продержится дольше.
func (g *Game) burEscape(which int) bool {
	pad := g.burPad[which]
	other := g.burPad[1-which]
	step := g.cols + 1
	longer := func(n int) bool { return n != other && g.counter[n] > g.counter[pad] }
	switch {
	case g.padX[pad] <= g.xMax-g.colStep && longer(pad+1):
		pad++
	case g.padX[pad] >= g.x0+g.colStep && longer(pad-1):
		pad--
	case g.padY[pad] >= g.yTop+g.rowStep && longer(pad+step):
		pad += step
	case g.padY[pad] <= g.yBot-g.rowStep && longer(pad-step):
		pad -= step
	default:
		return false
	}
	g.burPad[which] = pad
	return true
}

// caught — F_3b38: Бюрократ поймал Демократа.
func (g *Game) caught(which int) {
	if g.dead {
		return
	}
	p := g.burPad[which]
	g.scr.SetActivePage(0)
	g.sweep(10, 10000, 1, func(f int) int { return g.rand() % 5 * f })
	g.sweep(10000, 10, -1, func(f int) int { return g.rand() % 3 * f })
	g.nosound()
	g.playSample(g.data.Sounds[1])
	g.eraseCell(g.padX[p], g.padY[p])
	g.drawSprite(g.padX[p]-20, g.padY[p]-22, 0, g.data.Cross, false)
	g.delay(2000)
}

// burDrown — F_19ea: Бюрократ утонул вместе с листом; на его месте появляется «Авантюра».
func (g *Game) burDrown(pad, which int) {
	g.burErase(which)
	g.whistle()
	g.nosound()
	g.score += 1000
	g.drawScore()
	g.burActive[which] = false
	if which == 1 {
		g.burPad[1] = g.bur2Home
	} else {
		g.burPad[0] = g.goal
	}
	g.ripples(g.padX[pad], g.padY[pad])
	if g.adventure < g.npads {
		g.fillCircle(g.padX[g.adventure], g.padY[g.adventure], g.r, colPadFill, colPadBorder)
	}
	g.adventure = pad
	g.advGiven++
	g.counter[pad] = g.floatT[pad]
	g.padSize[pad] = g.r
	g.fillCircle(g.padX[pad], g.padY[pad], g.r, colPadFill, colPadBorder)
	g.drawBall(g.padX[pad], g.padY[pad], ballAdventure)
}

// goalShift — F_48d3 (с 10-го этапа): на случайном листе появляется
// «Прогрессивный налог», а монета этапа перепрыгивает в угол,
// противоположный Демократу.
func (g *Game) goalShift() {
	if g.rnd100()%21 != 0 {
		return
	}
	if g.rand()%(350-(g.level-10)*10) != 0 {
		return
	}
	p := g.rnd100() % (g.npads - 1)
	q := p
	// в оригинале проверяется goods[p], а не goods[q] — повторяем
	free := func(q int) bool { return !g.goods[p] && !g.tax[q] && q != g.cur }
	if g.npads/2 > p {
		for q < g.npads-1 && !free(q) {
			q++
		}
	} else {
		for q > 0 && !free(q) {
			q--
		}
	}
	g.tax[q] = true
	g.currency[q] = false
	g.counter[q] = g.floatT[q]
	g.padSize[q] = g.r
	g.fillCircle(g.padX[q], g.padY[q], g.r, colPadFill, colPadBorder)
	switch {
	case q == g.adventure:
		g.tax[q] = false
		g.drawBall(g.padX[q], g.padY[q], ballAdventure)
	case q == g.npads-1 || q == g.npads-1-g.cols || q == g.cols || q == 0:
		g.tax[q] = false
	default:
		g.drawBall(g.padX[q], g.padY[q], ballTax)
	}
	gx, gy := g.padX[g.goal], g.padY[g.goal]
	g.setWater()
	g.scr.Bar(gx-burHalfW, gy-burHalfH, gx+burHalfW, gy+burHalfH)
	g.fillCircle(gx, gy, g.r, colPadFill, colPadBorder)

	half := float64(g.cols+1) * 0.5
	col := float64(g.cur % (g.cols + 1))
	if float64(g.cur) > float64(g.npads)*0.5 {
		if col > half {
			g.goal = 0
		} else {
			g.goal = g.cols
		}
	} else {
		if col > half {
			g.goal = g.npads - 1 - g.cols
		} else {
			g.goal = g.npads - 1
		}
	}
	g.counter[g.goal] = g.floatT[g.goal]
	g.padSize[g.goal] = g.r
	g.fillCircle(g.padX[g.goal], g.padY[g.goal], g.r, colPadFill, colPadBorder)
	g.drawCoin(g.padX[g.goal], g.padY[g.goal])
}
