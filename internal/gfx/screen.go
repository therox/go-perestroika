// Package gfx — программная замена режима EGA 640x350x16 и графической
// библиотеки BGI, которыми пользовалась оригинальная игра.
//
// Как и в оригинале, есть две видеостраницы: игра рисует на «активной»
// (обычно невидимой странице 1) и переносит готовые области на видимую
// страницу 0 функцией CopyRect. Каждый примитив добавляет к счётчику Cost
// свою «стоимость» в инструкциях процессора, откалиброванную по оригиналу:
// по ней воспроизводится исходный темп игры.
package gfx

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// Размеры экрана EGA 640x350.
const (
	W    = 640
	H    = 350
	MaxX = W - 1
	MaxY = H - 1
)

// Стили заливки BGI.
const (
	EmptyFill = 0
	SolidFill = 1
	UserFill  = 12
)

// Выравнивание текста BGI.
const (
	LeftText   = 0
	CenterText = 1
	RightText  = 2
	BottomText = 0
	TopText    = 2
)

// DefaultPalette — палитра EGA по умолчанию (как после setallpalette в initgraph).
var DefaultPalette = [16]byte{0, 1, 2, 3, 4, 5, 20, 7, 56, 57, 58, 59, 60, 61, 62, 63}

// Screen — две видеостраницы и состояние графики в стиле BGI.
type Screen struct {
	mu sync.Mutex

	pages [2][]byte
	pal   [16]byte

	active  int
	color   byte
	fill    byte
	style   int
	pattern [8]byte
	cpx     int
	cpy     int
	justH   int
	justV   int

	cost float64
}

// Image — сохранённая прямоугольная область (аналог getimage/putimage).
type Image struct {
	W, H int
	Pix  []byte
}

// NewScreen создаёт экран с палитрой по умолчанию.
func NewScreen() *Screen {
	s := &Screen{pal: DefaultPalette, style: SolidFill, fill: 15, color: 15}
	for i := range s.pages {
		s.pages[i] = make([]byte, W*H)
	}
	return s
}

// TakeCost возвращает накопленную стоимость операций и обнуляет счётчик.
func (s *Screen) TakeCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cost
	s.cost = 0
	return c
}

// AddCost добавляет стоимость вне графических примитивов.
func (s *Screen) AddCost(c float64) {
	s.mu.Lock()
	s.cost += c
	s.mu.Unlock()
}

// SetActivePage выбирает страницу для рисования.
func (s *Screen) SetActivePage(p int) {
	s.mu.Lock()
	s.active = p & 1
	s.cost += 60
	s.mu.Unlock()
}

// ClearDevice очищает активную страницу и ставит текущую позицию в (0,0).
func (s *Screen) ClearDevice() {
	s.mu.Lock()
	pg := s.pages[s.active]
	for i := range pg {
		pg[i] = 0
	}
	s.cpx, s.cpy = 0, 0
	s.cost += 47528
	s.mu.Unlock()
}

// SetColor задаёт цвет линий и текста.
func (s *Screen) SetColor(c int) {
	s.mu.Lock()
	s.color = byte(c & 15)
	s.cost += 40
	s.mu.Unlock()
}

// SetFillStyle задаёт стиль и цвет заливки.
func (s *Screen) SetFillStyle(style, c int) {
	s.mu.Lock()
	s.style = style
	s.fill = byte(c & 15)
	s.cost += 60
	s.mu.Unlock()
}

// SetFillPattern задаёт пользовательский шаблон 8x8 и цвет заливки.
func (s *Screen) SetFillPattern(p [8]byte, c int) {
	s.mu.Lock()
	s.pattern = p
	s.fill = byte(c & 15)
	s.style = UserFill
	s.cost += 80
	s.mu.Unlock()
}

// SetTextJustify задаёт выравнивание текста 8x8.
func (s *Screen) SetTextJustify(h, v int) {
	s.mu.Lock()
	s.justH, s.justV = h, v
	s.mu.Unlock()
}

// TextJustify возвращает текущее выравнивание текста.
func (s *Screen) TextJustify() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.justH, s.justV
}

// SetPalette задаёт цвет EGA (0..63) для индекса палитры.
func (s *Screen) SetPalette(i int, c byte) {
	s.mu.Lock()
	s.pal[i&15] = c & 63
	s.mu.Unlock()
}

// ResetPalette восстанавливает палитру по умолчанию.
func (s *Screen) ResetPalette() {
	s.mu.Lock()
	s.pal = DefaultPalette
	s.mu.Unlock()
}

func (s *Screen) pset(x, y int, c byte) {
	if x < 0 || y < 0 || x > MaxX || y > MaxY {
		return
	}
	s.pages[s.active][y*W+x] = c
}

// fillPixel возвращает цвет заливки в точке с учётом шаблона,
// привязанного к экранным координатам, как в BGI.
func (s *Screen) fillPixel(x, y int) byte {
	switch s.style {
	case EmptyFill:
		return 0
	case UserFill:
		if s.pattern[y&7]&(0x80>>uint(x&7)) != 0 {
			return s.fill
		}
		return 0
	default:
		return s.fill
	}
}

// Bar заливает прямоугольник без рамки.
func (s *Screen) Bar(l, t, r, b int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bar(l, t, r, b)
}

func (s *Screen) bar(l, t, r, b int) {
	if l > r {
		l, r = r, l
	}
	if t > b {
		t, b = b, t
	}
	h := b - t + 1
	bytes := ((r >> 3) - (l >> 3) + 1) * h
	if s.style == UserFill {
		s.cost += 320 + 57*float64(h) + 3.9*float64(bytes)
	} else {
		s.cost += 240 + 55*float64(h) + float64(bytes)
	}
	l, t, r, b = max(l, 0), max(t, 0), min(r, MaxX), min(b, MaxY)
	pg := s.pages[s.active]
	for y := t; y <= b; y++ {
		row := pg[y*W:]
		for x := l; x <= r; x++ {
			row[x] = s.fillPixel(x, y)
		}
	}
}

// Line рисует отрезок текущим цветом.
func (s *Screen) Line(x1, y1, x2, y2 int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.line(x1, y1, x2, y2)
}

func (s *Screen) line(x1, y1, x2, y2 int) {
	dx, dy := abs(x2-x1), abs(y2-y1)
	s.cost += 218 + 10*float64(max(dx, dy))
	sx, sy := 1, 1
	if x2 < x1 {
		sx = -1
	}
	if y2 < y1 {
		sy = -1
	}
	if dx >= dy {
		e := dx / 2
		y := y1
		for x := x1; ; x += sx {
			s.pset(x, y, s.color)
			if x == x2 {
				break
			}
			e -= dy
			if e < 0 {
				y += sy
				e += dx
			}
		}
	} else {
		e := dy / 2
		x := x1
		for y := y1; ; y += sy {
			s.pset(x, y, s.color)
			if y == y2 {
				break
			}
			e -= dx
			if e < 0 {
				x += sx
				e += dy
			}
		}
	}
}

// Rectangle рисует рамку прямоугольника.
func (s *Screen) Rectangle(l, t, r, b int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.line(l, t, r, t)
	s.line(r, t, r, b)
	s.line(r, b, l, b)
	s.line(l, b, l, t)
}

// MoveTo перемещает текущую позицию.
func (s *Screen) MoveTo(x, y int) {
	s.mu.Lock()
	s.cpx, s.cpy = x, y
	s.cost += 40
	s.mu.Unlock()
}

// LineTo рисует отрезок от текущей позиции и перемещает её.
func (s *Screen) LineTo(x, y int) {
	s.mu.Lock()
	s.line(s.cpx, s.cpy, x, y)
	s.cpx, s.cpy = x, y
	s.mu.Unlock()
}

// CP возвращает текущую позицию.
func (s *Screen) CP() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cpx, s.cpy
}

// FillEllipse рисует закрашенный эллипс с рамкой текущего цвета.
func (s *Screen) FillEllipse(x, y, rx, ry int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cost += 2100 + 1250*float64(rx)
	rows, ok := filledEllipses[ellipseKey{rx, ry}]
	if !ok {
		rows = approxEllipse(rx, ry)
	}
	for dy, r := range rows {
		if r.W < 0 {
			continue
		}
		for _, yy := range [2]int{y - dy, y + dy} {
			for dx := -int(r.W); dx <= int(r.W); dx++ {
				if abs(dx) > int(r.W)-int(r.B) {
					s.pset(x+dx, yy, s.color)
				} else {
					s.pset(x+dx, yy, s.fillPixel(x+dx, yy))
				}
			}
			if dy == 0 {
				break
			}
		}
	}
}

// Ellipse рисует контур полного эллипса текущим цветом.
func (s *Screen) Ellipse(x, y, rx, ry int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cost += 1170 + 553*float64(rx)
	rows, ok := outlineEllipses[ellipseKey{rx, ry}]
	if !ok {
		rows = approxEllipse(rx, ry)
	}
	for dy, r := range rows {
		if r.W < 0 {
			continue
		}
		for _, yy := range [2]int{y - dy, y + dy} {
			for k := 0; k < int(r.B); k++ {
				s.pset(x-int(r.W)+k, yy, s.color)
				s.pset(x+int(r.W)-k, yy, s.color)
			}
			if dy == 0 {
				break
			}
		}
	}
}

// approxEllipse строит форму эллипса для размеров, отсутствующих в эталонной таблице.
func approxEllipse(rx, ry int) []ellipseRow {
	rows := make([]ellipseRow, ry+1)
	prev := -1
	for dy := ry; dy >= 0; dy-- {
		t := 1.0
		if ry > 0 {
			t = 1 - float64(dy*dy)/float64(ry*ry)
		}
		w := int(float64(rx)*math.Sqrt(math.Max(t, 0)) + 0.5)
		b := 1
		if prev >= 0 && w-prev > 1 {
			b = w - prev
		}
		if dy == ry {
			b = w + 1
		}
		rows[dy] = ellipseRow{int8(w), int8(min(b, w+1))}
		prev = w
	}
	return rows
}

// OutTextXY выводит строку шрифтом 8x8 (CP866) текущим цветом с учётом выравнивания.
func (s *Screen) OutTextXY(x, y int, str []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := 8 * len(str)
	switch s.justH {
	case CenterText:
		// так центрирует BGI: на пиксель правее середины
		x -= w/2 - 1
	case RightText:
		x -= w
	}
	switch s.justV {
	case CenterText:
		y -= 4
	case BottomText:
		y -= 7
	}
	s.cost += 300 + 450*float64(len(str))
	for i, ch := range str {
		gx := x + i*8
		// BGI не выводит символ, не помещающийся целиком в окно
		if gx < 0 || gx+7 > MaxX || y < 0 || y+7 > MaxY {
			continue
		}
		for row := 0; row < 8; row++ {
			bits := font8x8[int(ch)*8+row]
			for col := 0; col < 8; col++ {
				if bits&(0x80>>uint(col)) != 0 {
					s.pset(gx+col, y+row, s.color)
				}
			}
		}
	}
}

// GetImage сохраняет область активной страницы.
func (s *Screen) GetImage(l, t, r, b int) *Image {
	s.mu.Lock()
	defer s.mu.Unlock()
	img := &Image{W: r - l + 1, H: b - t + 1}
	img.Pix = make([]byte, img.W*img.H)
	pg := s.pages[s.active]
	for y := 0; y < img.H; y++ {
		for x := 0; x < img.W; x++ {
			px, py := l+x, t+y
			if px >= 0 && py >= 0 && px <= MaxX && py <= MaxY {
				img.Pix[y*img.W+x] = pg[py*W+px]
			}
		}
	}
	s.cost += 200 + 4*float64(len(img.Pix))/8
	return img
}

// PutImage восстанавливает область (режим COPY_PUT).
func (s *Screen) PutImage(l, t int, img *Image) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for y := 0; y < img.H; y++ {
		for x := 0; x < img.W; x++ {
			s.pset(l+x, t+y, img.Pix[y*img.W+x])
		}
	}
	s.cost += 200 + 4*float64(len(img.Pix))/8
}

// Sprite — изображение с маской прозрачности.
type Sprite interface {
	Size() (w, h int)
	At(x, y int) (c byte, opaque bool)
}

// DrawSprite выводит спрайт на страницу page с прозрачностью по маске.
// При aligned=true координата x выравнивается вниз на границу байта,
// как в ассемблерной процедуре оригинала.
func (s *Screen) DrawSprite(x, y, page int, spr Sprite, aligned bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, h := spr.Size()
	if aligned {
		x &^= 7
		s.cost += 300 + 6*float64(w*h)
	} else if x&7 == 0 {
		s.cost += 300 + 6*float64(w*h)
	} else {
		s.cost += 300 + 16*float64(w*h)
	}
	pg := s.pages[page&1]
	for yy := 0; yy < h; yy++ {
		py := y + yy
		if py < 0 || py > MaxY {
			continue
		}
		for xx := 0; xx < w; xx++ {
			px := x + xx
			if px < 0 || px > MaxX {
				continue
			}
			if c, ok := spr.At(xx, yy); ok {
				pg[py*W+px] = c
			}
		}
	}
}

// CopyRect копирует прямоугольник со страницы src на страницу dst.
// Горизонтальные границы выравниваются на байты (8 пикселей), как в оригинале.
func (s *Screen) CopyRect(x1, y1, x2, y2, src, dx, dy, dst int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x1 &^= 7
	x2 |= 7
	// отсечение источника экраном
	cx1, cy1 := max(x1, 0), max(y1, 0)
	cx2, cy2 := min(x2, MaxX), min(y2, MaxY)
	if cx2 < cx1 || cy2 < cy1 {
		return
	}
	dx = (dx + cx1 - x1) &^ 7
	dy = dy + cy1 - y1
	w, h := cx2-cx1+1, cy2-cy1+1
	s.cost += 194 + 11*float64(h) + float64(w/8*h)
	sp, dp := s.pages[src&1], s.pages[dst&1]
	tmp := make([]byte, w)
	for y := 0; y < h; y++ {
		sy, ty := cy1+y, dy+y
		if ty < 0 || ty > MaxY {
			continue
		}
		copy(tmp, sp[sy*W+cx1:sy*W+cx1+w])
		for x := 0; x < w; x++ {
			tx := dx + x
			if tx >= 0 && tx <= MaxX {
				dp[ty*W+tx] = tmp[x]
			}
		}
	}
}

// PutPixels заполняет активную страницу готовым изображением 640x350.
func (s *Screen) PutPixels(pix []byte) {
	s.mu.Lock()
	copy(s.pages[s.active], pix)
	s.cost += 60000
	s.mu.Unlock()
}

// EGAColor переводит 6-битный цвет EGA (rgbRGB) в RGB.
func EGAColor(c byte) color.RGBA {
	v := func(hi, lo byte) uint8 { return 0xAA*hi + 0x55*lo }
	return color.RGBA{
		R: v((c>>2)&1, (c>>5)&1),
		G: v((c>>1)&1, (c>>4)&1),
		B: v(c&1, (c>>3)&1),
		A: 0xFF,
	}
}

// Frame — снимок видимой страницы: индексы цветов и палитра.
type Frame struct {
	Pix []byte
	Pal [16]byte
}

// NewFrame создаёт пустой кадр.
func NewFrame() *Frame { return &Frame{Pix: make([]byte, W*H)} }

// Capture копирует видимую страницу 0 и палитру в кадр f.
func (s *Screen) Capture(f *Frame) {
	s.mu.Lock()
	copy(f.Pix, s.pages[0])
	f.Pal = s.pal
	s.mu.Unlock()
}

// RGBA переводит кадр в RGBA (буфер W*H*4).
func (f *Frame) RGBA(dst []byte) {
	var lut [16][4]byte
	for i, c := range f.Pal {
		rgb := EGAColor(c)
		lut[i] = [4]byte{rgb.R, rgb.G, rgb.B, 0xFF}
	}
	for i, ci := range f.Pix {
		copy(dst[i*4:i*4+4], lut[ci&15][:])
	}
}

// Snapshot возвращает изображение указанной страницы (для отладки и тестов).
func (s *Screen) Snapshot(page int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	s.mu.Lock()
	defer s.mu.Unlock()
	pg := s.pages[page&1]
	for i, ci := range pg {
		c := EGAColor(s.pal[ci&15])
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = c.R, c.G, c.B, 0xFF
	}
	return img
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
