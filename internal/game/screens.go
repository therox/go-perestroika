package game

import (
	"fmt"
	"strconv"

	"github.com/therox/go-perestroika/internal/dat"
	"github.com/therox/go-perestroika/internal/gfx"
)

// gamePalette — изменения палитры, которые делает F_5203.
var gamePalette = [][2]byte{
	{8, 0x2c}, {12, 0x34}, {5, 0x26}, {6, 6}, {2, 0x0e}, {10, 0x12},
	{9, 9}, {1, 0x31}, {3, 0x2b}, {11, 0x13}, {7, 0x2e},
}

// Palette возвращает полную палитру игры (цвета EGA 0..63).
func Palette() [16]byte {
	p := gfx.DefaultPalette
	for _, e := range gamePalette {
		p[e[0]] = e[1]
	}
	return p
}

// setGamePalette — F_5203: палитра игры.
func (g *Game) setGamePalette() {
	for _, p := range gamePalette {
		g.scr.SetPalette(int(p[0]), p[1])
	}
}

// drawMainScreen — F_424d: заставка с подсказкой, титрами и легендой.
func (g *Game) drawMainScreen() {
	const mx, my = gfx.MaxX, gfx.MaxY
	g.scr.SetColor(0)
	g.scr.ClearDevice()
	g.scr.SetFillStyle(gfx.SolidFill, 4)
	g.scr.Bar(0, 0, mx, my)
	// панель легенды
	g.scr.SetFillStyle(gfx.SolidFill, colWater)
	g.scr.Rectangle(mx/2-58, 17, mx-18, my-13)
	g.scr.Rectangle(mx/2-59, 16, mx-19, my-14)
	g.scr.Bar(mx/2-60, 15, mx-20, my-15)
	g.scr.Rectangle(mx/2-60, 15, mx-20, my-15)
	// панель титров
	g.scr.SetFillStyle(gfx.SolidFill, colPadFill)
	g.scr.Rectangle(22, 102, mx/2-78, my-13)
	g.scr.Rectangle(21, 101, mx/2-79, my-14)
	g.scr.Bar(20, 100, mx/2-80, my-15)
	g.scr.Rectangle(20, 100, mx/2-80, my-15)

	g.scr.SetColor(14)
	g.rusText(130, 30, "Lkz buhs bcgjkmpeqnt") // «Для игры используйте»
	g.rusText(130, 50, "CNHTKRB REHCJHF")      // «СТРЕЛКИ КУРСОРА»
	g.rusText(130, 70, "TCR - Gfepf d buht")   // «ЕСК - Пауза в игре»
	g.drawSprite(mx/4-30-democratHalf, 120-democratHalf+2, 0, g.data.Democrat[2], false)
	g.drawSprite(mx/4-105-democratHalf, 120-democratHalf+2, 0, g.data.Democrat[1], false)
	g.drawSprite(mx/4+45-democratHalf, 120-democratHalf+2, 0, g.data.Democrat[3], false)

	g.scr.SetColor(10)
	g.rusText(mx/4-30, 180, "Dthcbz  2&2 ^ 1990 u&") // «Версия 2.2, 1990 г.»
	g.rusText(mx/4-30, 200, "Y &  Crhbgrby")         // «Н. Скрипкин»
	g.scr.SetColor(11)
	g.rusText(mx/4-30, 230, "Rjycekmnfwbb%")            // «Консультации:»
	g.rusText(mx/4-30, 250, "L & Xbrby")                // «Д. Чикин»
	g.rusText(mx/4-30, 270, "Cgtwbfkmyst 'aatrns%")     // «Специальные эффекты:»
	g.rusText(mx/4-30, 290, "V & Kj[fyjd  D & Vehjujd") // «М. Лоханов  В. Мурогов»
	g.rusText(mx/4-30, 310, "D & Kt,tltd  G & Uehdbx ") // «В. Лебедев  П. Гурвич»

	// легенда
	g.fillCircle(mx/2, 50, 20, colPadFill, colPadBorder)
	g.scr.SetColor(14)
	g.rusText(mx/2+152, 50, "- Pfrjys b Gjcnfyjdktybz") // «- Законы и Постановления»
	g.drawSprite(mx/2-democratHalf, 95-democratHalf+2, 0, g.data.Democrat[2], false)
	g.rusText(mx/2+130, 95, "- L t v j r h f n") // «- Д е м о к р а т»
	g.drawSprite(mx/2-burHalfW/2-6, 130-burHalfH/2-4, 0, g.data.Bureaucrat[0], false)
	g.rusText(mx/2+130, 130, "- < . h j r h f n") // «- Б ю р о к р а т»
	g.drawBall(mx/2, 165, ballGoods)
	g.scr.SetColor(14)
	g.rusText(mx/2+168, 165, "- Njdfhs yfhjlyjuj gjnht,ktybz") // «- Товары народного потребления»
	g.drawBall(mx/2, 200, ballCurrency)
	g.scr.SetColor(14)
	g.rusText(mx/2+133, 200, "- Dfk.nyst jgthfwbb") // «- Валютные операции»
	g.drawBall(mx/2, 235, ballTax)
	g.scr.SetColor(14)
	g.rusText(mx/2+138, 235, "- Ghjuhtccbdysq yfkju") // «- Прогрессивный налог»
	g.drawBall(mx/2, 270, ballAdventure)
	g.scr.SetColor(14)
	g.rusText(mx/2+130, 270, "- F d f y n . h f") // «- А в а н т ю р а»
	g.drawSprite(mx/2-burHalfW/2-2, 305-burHalfH/2, 0, g.data.Coins[24], false)
	g.rusText(mx/2+138, 305, "- \"nfgs <jkmijuj Genb") // «- Этапы Большого Пути»
	g.work(0)
	g.showHiScores()
}

// drawWindow — оранжевое окно с тенью, как в оригинале.
func (g *Game) drawWindow(x1, y1, x2, y2, fill, color int) {
	g.scr.SetFillStyle(gfx.SolidFill, fill)
	g.scr.SetColor(color)
	g.scr.Rectangle(x1+2, y1+2, x2, y2)
	g.scr.Rectangle(x1+1, y1+1, x2-1, y2-1)
	g.scr.Bar(x1, y1, x2-2, y2-2)
	g.scr.Rectangle(x1, y1, x2-2, y2-2)
}

// showHiScores — F_2f5e: окно «ЛУЧШИЕ ДЕМОКРАТЫ»; ждёт клавишу и восстанавливает фон.
func (g *Game) showHiScores() {
	x1 := gfx.MaxX/8 - 45
	y1 := gfx.MaxY/6 - 43
	g.flushKeys()
	jh, jv := g.scr.TextJustify()
	const charW, lineH = 8, 12
	g.scr.SetTextJustify(gfx.LeftText, gfx.CenterText)
	x2 := (21+2)*charW + x1
	y2 := lineH*12 + y1
	saved := g.scr.GetImage(x1, y1, x2, y2)
	g.drawWindow(x1, y1, x2, y2, 7, 0)
	g.scr.SetColor(0)
	tx := ftol(float64(x1) + charW*1.5)
	ty := y1 + lineH
	g.scr.SetColor(14)
	g.rusText(tx+79, ty-4, "KEXIBT LTVJRHFNS") // «ЛУЧШИЕ ДЕМОКРАТЫ»
	ty += lineH
	for i, h := range g.hiScores {
		switch {
		case i < 1:
			g.scr.SetColor(4)
		case i < 3:
			g.scr.SetColor(1)
		default:
			g.scr.SetColor(0)
		}
		var line []byte
		if i < len(g.hiScores)-1 {
			line = append(line, ' ')
		}
		line = append(line, strconv.Itoa(i+1)...)
		line = append(line, "  "...)
		line = append(line, cstr(h.Name[:])...)
		line = append(line, "  "...)
		line = append(line, strconv.FormatInt(int64(h.Score), 10)...)
		g.scr.OutTextXY(tx, ty, line)
		ty += lineH
	}
	g.work(0)
	g.waitKey()
	g.scr.PutImage(x1, y1, saved)
	g.scr.SetTextJustify(jh, jv)
	g.work(0)
}

// cstr возвращает байты до завершающего нуля.
func cstr(b []byte) []byte {
	for i, c := range b {
		if c == 0 {
			return b[:i]
		}
	}
	return b
}

// checkHiScore — F_2b7a: если результат попадает в таблицу, спросить имя и вставить запись.
func (g *Game) checkHiScore() bool {
	if g.hiScores[len(g.hiScores)-1].Score >= g.score {
		return false
	}
	name := g.askName()
	for i := range g.hiScores {
		if g.hiScores[i].Score > g.score {
			continue
		}
		copy(g.hiScores[i+1:], g.hiScores[i:len(g.hiScores)-1])
		g.hiScores[i] = dat.HiScore{Score: g.score, Name: name}
		break
	}
	return true
}

// askName — F_2c8a: окно «Введи свое имя».
func (g *Game) askName() [9]byte {
	const prompt = "Ddtlb cdjt bvz %          " // «Введи свое имя :»
	x1 := gfx.MaxX/3 - 5
	y1 := gfx.MaxY*3/4 + 20
	g.flushKeys()
	jh, jv := g.scr.TextJustify()
	const charW, charH = 8, 16
	g.scr.SetTextJustify(gfx.LeftText, gfx.CenterText)
	x2 := (len(prompt)+3)*charW + x1
	y2 := charH*2 + y1
	saved := g.scr.GetImage(x1, y1, x2, y2)
	g.drawWindow(x1, y1, x2, y2, 7, 0)
	g.scr.SetColor(0)
	tx := ftol(float64(x1) + charW*1.5 + 90)
	ty := y1 + charH
	g.rusText(tx, ty, prompt)
	name := g.inputName((len(prompt)-9)*charW+x1, ty)
	g.scr.PutImage(x1, y1, saved)
	g.scr.SetTextJustify(jh, jv)
	g.work(0)
	return name
}

// inputName — F_3325: ввод имени до 8 символов.
func (g *Game) inputName(x, y int) [9]byte {
	const step, maxLen = 10, 8
	var buf [9]byte
	blank := [9]byte{' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', 0}
	n := 0
	g.scr.SetColor(4)
	for {
		g.work(0)
		key := g.getKey()
		if key == 13 {
			break
		}
		if n >= maxLen && key != 8 {
			g.bell()
			continue
		}
		if key == 27 {
			return blank
		}
		if key == 0x20 || key > 0x100 {
			continue
		}
		if key == 8 {
			if n == 0 {
				continue
			}
			n--
			g.scr.SetFillStyle(gfx.SolidFill, 7)
			g.scr.Bar(x+(n+1)*step, y-5, x+(n+2)*step, y+5)
			continue
		}
		ch := byte(key)
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		buf[n] = ch
		g.scr.OutTextXY(x+n*step+step, y, []byte{ch})
		n++
	}
	if n == 0 {
		return blank
	}
	for i := n; i < maxLen; i++ {
		buf[i] = ' '
	}
	buf[maxLen] = 0
	return buf
}

// bell — printf("\a"): короткий сигнал BIOS.
func (g *Game) bell() {
	g.sound(896)
	g.delay(120)
	g.nosound()
}

// showStats — F_4de7: статистика игры в окне конца игры.
func (g *Game) showStats(x1, y1, x2 int) {
	dt := float32(g.endTime - g.startTime)
	h := ftol(float64(dt / 3600))
	rem := dt - float32(h)*3600
	m := ftol(float64(rem / 60))
	rem -= float32(m) * 60
	s := ftol(float64(rem))
	cx := (x2 + x1) / 2
	g.scr.SetColor(14)
	g.rusText(cx, y1+15, "Cnfnbcnbrf Dfitq buhs%") // «Статистика Вашей игры:»
	g.scr.SetColor(0)
	g.rusText(cx, y1+35, fmt.Sprintf("Ghjljk;bntkmyjcnm %d xfc& %d vby& %d ctr&", h, m, s)) // «Продолжительность ... час. ... мин. ... сек.»
	g.rusText(x1+100, y1+55, "Lfyj%")                                                       // «Дано:»
	g.rusText(x1+200, y1+55, "Htfkbpjdfyj%")                                                // «Реализовано:»
	g.scr.SetTextJustify(gfx.CenterText, gfx.CenterText)
	num := func(x, y, v int) { g.scr.OutTextXY(x, y, []byte(strconv.Itoa(v))) }
	g.scr.SetColor(colGoodsFill)
	num(x1+100, y1+75, g.goodsGiven)
	num(x1+200, y1+75, g.goodsTaken)
	g.scr.SetColor(colCurFill)
	num(x1+100, y1+90, g.curGiven)
	num(x1+200, y1+90, g.curTaken)
	g.scr.SetColor(colAdvFill)
	num(x1+100, y1+105, g.advGiven)
	num(x1+200, y1+105, g.advTaken)
	g.scr.SetColor(14)
	g.rusText(x1+255, y1+105, "Bp yb[%")                                           // «Из них:»
	g.rusText(x1+79, y1+125, fmt.Sprintf(":bpym ltvjrhfnf - %d", g.advOutcome[0])) // «Жизнь демократа»
	g.scr.SetColor(0)
	g.rusText(x1+240, y1+125, fmt.Sprintf("Cvthnm ltvjrhfnf - %d", g.advOutcome[1])) // «Смерть демократа»
	g.scr.SetColor(14)
	g.rusText(x1+100, y1+140, fmt.Sprintf("Vfnthbfkmyjt gjjohtybt - %d", g.advOutcome[2])) // «Материальное поощрение»
	g.rusText(x1+81, y1+155, fmt.Sprintf("Cvthnm ,.hjrhfnf - %d", g.advOutcome[3]))        // «Смерть бюрократа»
	g.scr.SetColor(0)
	g.rusText(x1+240, y1+155, fmt.Sprintf(":bpym ,.hjrhfnf - %d", g.advOutcome[4])) // «Жизнь бюрократа»
	g.drawBall(x1+30, y1+75, ballGoods)
	g.drawBall(x1+30, y1+90, ballCurrency)
	g.drawBall(x1+30, y1+105, ballAdventure)
	g.scr.SetColor(0)
	g.work(0)
	g.flushKeys()
	g.waitKey()
}

// gameOver — F_257c: итоги и вопрос «Будем ещё ПЕРЕСТРАИВАТЬСЯ (1/0)?».
// Возвращает true, если игрок выбрал продолжить.
func (g *Game) gameOver() bool {
	const mx, my = gfx.MaxX, gfx.MaxY
	x1, y1 := mx*2/8, my*2/8
	x2, y2 := mx*6/8, my*6/8
	g.scr.SetActivePage(0)
	g.drawWindow(x1, y1, x2, y2, 7, 0)
	g.endTime = g.now()
	g.showStats(x1, y1, x2-2)
	if g.checkHiScore() {
		if g.store != nil {
			_ = g.store.SaveHiScores(g.hiScores)
		}
		g.showHiScores()
	}
	g.scr.SetFillStyle(gfx.SolidFill, 7)
	g.scr.Bar(x1, y1, x2-2, y2-2)
	g.drawSprite(mx/4+55, my/4+20, 0, g.data.Condolence, true)
	g.scr.SetActivePage(1)
	g.scr.Bar(x1+8, my/2+50, x2-40, my/2+70)
	g.rusText(mx/2, my/2+60, "<eltv to] GTHTCNHFBDFNMCZ ( 1 / 0 ) ?") // «Будем ещё ПЕРЕСТРАИВАТЬСЯ ( 1 / 0 ) ?»
	g.work(0)

	// ждать «1» или «0» не дольше n опросов клавиатуры
	wait := func(n int) (int, bool) {
		for k := 0; k < n; k++ {
			if key := g.pollKey(); key == '1' || key == '0' {
				return key, true
			}
		}
		return 0, false
	}
	frame, blink := 0, 3
	key := 0
	for {
		// В оригинале перед поклоном (кадр 0) паузы не было: он рисовался сразу
		// после второго шага, и тот мелькал на долю секунды. Здесь пауза перед
		// каждым кадром одинаковая, чтобы танец шёл ровно.
		var done bool
		if key, done = wait(1100); done {
			break
		}
		// марширующие Бюрократы
		g.scr.Bar(x1+1, my/2, x2-10, my/2+40)
		for _, dx := range [5]int{70, 110, 150, 190, 230} {
			g.drawSprite(x1+dx, my/2+5, 1, g.data.Bureaucrat[g.burFrame[0]], true)
		}
		g.nextFrame(0)
		g.scr.CopyRect(mx/8+x1-15, my/2, x2-40, my/2+40, 1, mx/8+x1-15, my/2, 0)
		g.work(0)
		if frame != 0 {
			g.beep2()
		} else {
			if key, done = wait(300); done {
				break
			}
			g.playSample(g.data.Sounds[dat.SndChaCha])
		}
		frame++
		if frame > 2 {
			frame = 0
			if blink == 0 {
				blink = 2
				// Оригинал показывал вопрос только на время отдельной паузы
				// (2000 опросов) и сразу стирал его, так что большую часть времени
				// не было видно, что нажимать. Здесь вопрос, появившись, остаётся
				// на экране, а лишняя пауза не сбивает ритм танца.
				g.scr.CopyRect(x1+28, my/2+50, x2-40, my/2+70, 1, x1+28, my/2+50, 0)
				g.work(0)
			} else {
				blink--
			}
		}
	}
	g.scr.SetActivePage(0)
	return key == '1'
}
