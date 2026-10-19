package game

import "github.com/therox/go-perestroika/internal/gfx"

// genLevel — F_080d: сетка листьев и расстановка бонусов. Возвращает число листьев.
func (g *Game) genLevel() int {
	g.r = gfx.MaxX / g.cols / 3
	rows := g.cols * 350 / 640
	g.x0 = 2 * g.r
	g.yBot = ftol(gfx.MaxY - float64(2*g.r)*g.aspect)
	g.xMax = gfx.MaxX - 2*g.r
	g.yTop = ftol(float64(2*g.r)*g.aspect + 24)
	g.colStep = (g.xMax - g.x0) / g.cols
	g.rowStep = (g.yBot - g.yTop) / rows
	g.xMax = g.colStep*g.cols + g.x0
	g.yTop = g.yBot - g.rowStep*rows

	i := 0
	for y := g.yBot; y >= g.yTop; y -= g.rowStep {
		for x := g.x0; x <= g.xMax; x += g.colStep {
			g.padX[i], g.padY[i] = x, y
			if g.level > 1 {
				a := g.rnd100()
				if (a+g.rand()%100)%50/g.level == 0 {
					g.goods[i] = true
					g.goodsGiven++
				} else {
					g.goods[i] = false
				}
				b := g.rnd100()
				if (b+g.rand()%100)%190/g.level == 0 && !g.goods[i] {
					g.currency[i] = true
					g.curGiven++
				} else {
					g.currency[i] = false
				}
			} else {
				g.goods[i] = false
				g.currency[i] = false
			}
			g.tax[i] = false
			i++
		}
	}
	if g.level == 1 {
		g.goodsGiven, g.curGiven = 0, 0
	}
	g.work(float64(i) * 400)
	return i
}

// initPads — F_1199: время жизни листьев на плаву и под водой.
func (g *Game) initPads() {
	for i := 1; i < g.npads; i++ {
		a := g.rnd100()
		g.sinkT[i] = -int32((a+g.rand()%100)%(30-g.level/2) + 1)
		b := g.rnd100()
		g.floatT[i] = int32((b+g.rand()%100)%(30-g.level) + 5)
		rv := g.rand()
		if rv%g.rnd100()%2 != 0 {
			g.counter[i] = g.floatT[i]
		} else {
			g.counter[i] = g.sinkT[i]
		}
		g.padSize[i] = g.r
	}
	for _, p := range [2]int{g.goal, 0} {
		g.sinkT[p] = -10
		g.floatT[p] = 100
		g.counter[p] = 100
		g.padSize[p] = g.r
	}
	// листья у старта всначале на плаву и тонут ненадолго
	for _, p := range [5]int{1, 2, g.cols + 1, g.cols + 2, 2*g.cols + 2} {
		g.counter[p] = g.floatT[p]
		g.sinkT[p] /= 4
	}
	if g.burActive[1] {
		g.counter[g.bur2Home] = g.floatT[g.bur2Home]
	}
	g.counter[g.adventure] = g.floatT[g.adventure]
	g.work(float64(g.npads) * 600)
}

// drawLevel — F_09fa: полная отрисовка уровня.
func (g *Game) drawLevel() {
	g.scr.SetActivePage(1)
	g.drawBackground()
	for i := 0; i < g.npads; i++ {
		if g.counter[i] < 1 {
			continue
		}
		x, y := g.padX[i], g.padY[i]
		g.fillCircle(x, y, g.r, colPadFill, colPadBorder)
		if g.goods[i] {
			g.drawBall(x, y, ballGoods)
		}
		if g.tax[i] {
			g.drawBall(x, y, ballTax)
		}
		if i == g.adventure {
			g.drawBall(x, y, ballAdventure)
		}
		if g.currency[i] {
			g.drawBall(x, y, ballCurrency)
		}
	}
	g.drawDemocrat(g.padX[0], g.padY[0])
	g.drawCoin(g.padX[g.goal], g.padY[g.goal])
	g.burPad[0] = g.goal
	g.burPad[1] = g.bur2Home
	if g.burActive[0] {
		g.drawBureaucrat(0, false)
	}
	if g.burActive[1] {
		g.drawBureaucrat(1, false)
	}
	g.scr.CopyRect(0, 0, gfx.MaxX, gfx.MaxY, 1, 0, 0, 0)
	g.work(0)
}
