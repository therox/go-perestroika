package game

import (
	"strconv"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/gfx"
)

func (g *Game) drawSprite(x, y, page int, s *dat.Sprite, aligned bool) {
	g.scr.DrawSprite(x, y, page, s, aligned)
}

// rusText — F_3fc2: строка векторным шрифтом, центрированная по x.
// Строка задаётся в «латинской» кодировке оригинала (ЙЦУКЕН).
func (g *Game) rusText(x, y int, s string) {
	f := g.data.Font
	n := f.MaxIndex()
	width := 0
	for i := 0; i < len(s); i++ {
		idx := int(s[i]) - 0x21
		switch {
		case idx > n:
		case idx == -1:
			width += 6
		default:
			width += f.Width(idx)
		}
	}
	cx := ftol(float64(x) - float64(width)*0.5)
	cy := y + 3
	for i := 0; i < len(s); i++ {
		idx := int(s[i]) - 0x21
		if idx > n {
			continue
		}
		if idx == -1 {
			px, _ := g.scr.CP()
			cx = px + 6
			g.scr.MoveTo(cx, cy)
			continue
		}
		// F_412a: штрихи глифа от точки (cx, cy)
		g.scr.MoveTo(cx, cy)
		for _, st := range f.Glyph(idx) {
			if st.Draw {
				g.scr.LineTo(cx+st.DX, cy-st.DY)
			} else {
				g.scr.MoveTo(cx+st.DX, cy-st.DY)
			}
		}
		cx, cy = g.scr.CP()
		g.scr.AddCost(1000)
	}
	g.scr.AddCost(8000)
}

// fillCircle — F_07c2: лист (эллипс с учётом пропорций экрана).
func (g *Game) fillCircle(x, y, r, fill, border int) {
	g.scr.SetColor(border)
	g.scr.SetFillStyle(gfx.SolidFill, fill)
	g.scr.FillEllipse(x, y, r, ftol(float64(r)*g.aspect*0.7))
}

// drawBall — F_0cee: шарик-бонус с тенью и бликом.
func (g *Game) drawBall(x, y, kind int) {
	var border, fill int
	switch kind {
	case ballGoods:
		border, fill = colGoodsBorder, colGoodsFill
	case ballCurrency:
		border, fill = colCurBorder, colCurFill
	case ballTax:
		border, fill = colTaxBorder, colTaxFill
	case ballAdventure:
		border, fill = colAdvBorder, colAdvFill
	}
	ry := ftol(ballRadius * g.aspect)
	g.scr.SetFillStyle(gfx.SolidFill, 0)
	g.scr.SetColor(0)
	g.scr.FillEllipse(x+1, y, ballRadius, ry)
	g.scr.SetFillStyle(gfx.SolidFill, fill)
	g.scr.SetColor(border)
	g.scr.FillEllipse(x, y-1, ballRadius, ry)
	g.scr.SetColor(15)
	g.scr.Line(x, y-2, x, y-2)
	g.scr.Line(x-1, y-3, x-1, y-2)
}

// drawCoin — F_0e16: монета текущего этапа.
func (g *Game) drawCoin(x, y int) {
	c := g.data.Coins[g.level-1]
	g.drawSprite(x-burHalfW/2-2, y-burHalfH/2-5, 1, c, false)
}

// drawScore — F_0e5e: «Благосостояние» в облаке.
func (g *Game) drawScore() {
	g.drawSprite(541, 1, 1, g.data.Cloud, false)
	g.scr.SetColor(4)
	g.scr.OutTextXY(ftol(gfx.MaxX*0.925), 11, []byte(strconv.FormatInt(int64(g.score), 10)))
}

// drawLives — F_0ed5: «Соратники» — оставшиеся жизни.
func (g *Game) drawLives() {
	g.scr.SetFillStyle(gfx.SolidFill, colStatusBar)
	g.scr.Bar(ftol(gfx.MaxX*0.5+15), 0, ftol(gfx.MaxX*0.5+70), 22)
	for i := 0; i < g.lives-1; i++ {
		x := ftol(float64(i*20) + gfx.MaxX*0.5 + 25)
		g.drawSprite(x-democratHalf, 11-democratHalf+2, 1, g.data.Democrat[2], false)
	}
}

// sparkle — F_4c3b: блики на воде у кромки строки состояния.
func (g *Game) sparkle(y1, y2 int) {
	g.scr.SetFillPattern(waterPattern, colGoodsFill)
	g.scr.SetFillStyle(gfx.UserFill, colGoodsFill)
	g.scr.Bar(0, y1, gfx.MaxX, y2)
	g.scr.SetColor(11)
	h := y2 - y1 + 1
	for i := 0; i < 20; i++ {
		x := g.rand() % gfx.MaxX
		y := g.rand()%h + y1
		g.scr.Line(x, y, x+2, y)
	}
	g.work(400)
}

// setWater — заливка узором воды.
func (g *Game) setWater() {
	g.scr.SetFillPattern(waterPattern, colWater)
	g.scr.SetFillStyle(gfx.UserFill, colWater)
}

// eraseCell — F_1aeb: стереть лист (залить клетку водой).
func (g *Game) eraseCell(x, y int) {
	g.setWater()
	g.scr.Bar(x-g.r, y-g.r-2, x+g.r, y+g.r-2)
}

// drawDemocrat — F_0c1a: Демократ на листе; область сразу переносится на экран.
func (g *Game) drawDemocrat(x, y int) {
	c := g.cur
	if g.goods[c] || g.tax[c] || g.currency[c] || c == g.adventure {
		g.fillCircle(g.padX[c], g.padY[c], g.padSize[c], colPadFill, colPadBorder)
	}
	g.drawSprite(x-democratHalf, y-democratHalf-2, 1, g.data.Democrat[g.pose], false)
	g.scr.CopyRect(x-g.r, y-g.r-2, x+g.r, y+g.r-2, 1, x-g.r, y-g.r-2, 0)
}

// eraseDemocrat — F_10e2: убрать Демократа с прежнего листа.
func (g *Game) eraseDemocrat(x, y int) {
	g.setWater()
	g.scr.Bar(x-g.r, y-g.r-2, x+g.r, y+g.r-2)
	g.fillCircle(x, y, g.padSize[g.prev], colPadFill, colPadBorder)
	g.scr.CopyRect(x-g.r, y-g.r-2, x+g.r, y+g.r-2, 1, x-g.r, y-g.r-2, 0)
}

// drawBureaucrat рисует Бюрократа which на его листе текущим кадром.
func (g *Game) drawBureaucrat(which int, aligned bool) {
	p := g.burPad[which]
	g.drawSprite(g.padX[p]-burHalfW/2-6, g.padY[p]-burHalfH/2-9, 1,
		g.data.Bureaucrat[g.burFrame[which]], aligned)
}

// nextFrame переключает кадр анимации Бюрократа.
func (g *Game) nextFrame(which int) {
	g.burFrame[which]++
	if g.burFrame[which] > 7 {
		g.burFrame[which] = 0
	}
}

// drawBackground — F_05f6: вода, строка состояния, монеты пройденных этапов.
func (g *Game) drawBackground() {
	g.setWater()
	g.scr.Bar(0, 31, gfx.MaxX, gfx.MaxY)
	g.scr.SetFillStyle(gfx.SolidFill, colStatusBar)
	g.scr.Bar(0, 0, gfx.MaxX, 22)
	g.scr.SetColor(11)
	g.scr.Line(0, 23, gfx.MaxX, 23)
	g.scr.SetColor(10)
	g.rusText(gfx.MaxX/8-5, 10, ":BPYTYYSQ EHJDTYM%") // «ЖИЗНЕННЫЙ УРОВЕНЬ:»
	x := (gfx.MaxX/8 + 165) - (25-g.level)*4
	odd := g.level%2 != 0
	for i := 0; i < g.level-1; i++ {
		if g.level-2 == i {
			odd = false
		}
		y := 11 - burHalfH/2
		if odd {
			y++
		}
		g.drawSprite(x-burHalfW/2-2, y, 1, g.data.Coins[i], false)
		x -= 4
		odd = !odd
	}
	g.scr.SetColor(colStatusText)
	g.rusText(gfx.MaxX/2-20, 10, "CJHFNYBRB%")       // «СОРАТНИКИ:»
	g.rusText(gfx.MaxX/2+175, 10, "<KFUJCJCNJZYBT%") // «БЛАГОСОСТОЯНИЕ:»
	g.drawLives()
	g.drawScore()
	g.sparkle(24, 30)
}
