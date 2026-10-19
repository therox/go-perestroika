// Package dat разбирает файл данных оригинальной игры TOPPLER.DAT.
//
// Файл читался оригиналом строго последовательно, поэтому смещения блоков
// фиксированы:
//
//	0       158100  оцифрованный звук заставки (6-битный ШИМ, ~9322 Гц)
//	158100  65376   титульная картинка в формате PCX (640x350, 4 плоскости)
//	223476  ...     спрайты в собственном 5-плоскостном формате
//	250946  ...     пять звуковых эффектов
//	276446  2023    векторный шрифт с кириллицей
//	278469  130     таблица рекордов (10 записей по 13 байт)
package dat

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Смещения и размеры блоков в TOPPLER.DAT.
const (
	offIntro   = 0
	lenIntro   = 158100
	offTitle   = 158100
	lenTitle   = 65376
	offSprites = 223476
	offFont    = 276446
	lenFont    = 2023
	offScores  = 278469
	lenScores  = 130
	totalSize  = 278599

	// SampleRate — частота воспроизведения оцифрованного звука:
	// таймер программировался на 1193182/64 Гц, сэмпл выводился на каждом втором прерывании.
	SampleRate = 1193182.0 / 64 / 2
)

// Размеры блоков спрайтов и звуков (в порядке следования в файле).
var (
	spriteBlocks = []struct{ count, size, w, h int }{
		{4, 427, 27, 21},   // Демократ: 4 направления
		{25, 427, 31, 21},  // монеты «Этапов Большого Пути» 1..25
		{8, 607, 38, 24},   // Бюрократ: 8 кадров анимации
		{1, 1417, 43, 47},  // крест
		{1, 7427, 218, 53}, // надпись «Примите соболезнования»
		{1, 1387, 94, 23},  // облако
	}
	soundSizes = []int{2470, 7192, 4577, 3560, 7701}
)

// Sprite — изображение в индексах цветов EGA с маской прозрачности.
type Sprite struct {
	W, H   int
	Pix    []byte // индекс цвета 0..15 на пиксель
	Opaque []bool // true — пиксель непрозрачен
}

// Звуковые эффекты (номера в Data.Sounds).
const (
	SndGoods    = iota // «Товары народного потребления»
	SndCaught          // Демократа поймал Бюрократ
	SndCurrency        // «Валютные операции»
	SndTax             // «Прогрессивный налог»
	SndChaCha          // «ча-ча-ча» к танцу Бюрократов в конце игры
)

// HiScore — запись таблицы рекордов.
type HiScore struct {
	Score int32
	Name  [9]byte // CP866, дополнено пробелами до 8 символов, завершено нулём
}

// Data — все ресурсы игры.
type Data struct {
	Intro      []byte // звук заставки (значения ШИМ 1..64)
	Title      []byte // титульная картинка 640x350, индексы цветов
	Democrat   [4]*Sprite
	Coins      [25]*Sprite
	Bureaucrat [8]*Sprite
	Cross      *Sprite
	Condolence *Sprite
	Cloud      *Sprite
	Sounds     [5][]byte
	Font       *VFont
	HiScores   [10]HiScore
}

// ErrFormat — файл не похож на TOPPLER.DAT «Перестройки» версии 2.2.
var ErrFormat = errors.New("файл не похож на TOPPLER.DAT «Перестройки» версии 2.2")

// Parse разбирает содержимое TOPPLER.DAT.
func Parse(raw []byte) (*Data, error) {
	if len(raw) != totalSize {
		return nil, fmt.Errorf("%w: размер %d байт вместо %d", ErrFormat, len(raw), totalSize)
	}
	d := &Data{}
	d.Intro = pages(raw[offIntro : offIntro+lenIntro])

	title, err := decodePCX(raw[offTitle : offTitle+lenTitle])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	d.Title = title

	pos := offSprites
	var sprites []*Sprite
	for _, b := range spriteBlocks {
		for i := 0; i < b.count; i++ {
			s, err := decodeSprite(raw[pos : pos+b.size])
			if err != nil {
				return nil, fmt.Errorf("%w: спрайт по смещению %d: %v", ErrFormat, pos, err)
			}
			if s.W != b.w || s.H != b.h {
				return nil, fmt.Errorf("%w: спрайт по смещению %d размером %dx%d", ErrFormat, pos, s.W, s.H)
			}
			sprites = append(sprites, s)
			pos += b.size
		}
	}
	copy(d.Democrat[:], sprites[0:4])
	copy(d.Coins[:], sprites[4:29])
	copy(d.Bureaucrat[:], sprites[29:37])
	d.Cross, d.Condolence, d.Cloud = sprites[37], sprites[38], sprites[39]

	for i, n := range soundSizes {
		d.Sounds[i] = pages(raw[pos : pos+n])
		pos += n
	}
	if pos != offFont {
		return nil, fmt.Errorf("внутренняя ошибка разметки: шрифт ожидался по смещению %d, получено %d", offFont, pos)
	}
	d.Font, err = parseVFont(raw[offFont : offFont+lenFont])
	if err != nil {
		return nil, err
	}
	d.HiScores = ParseHiScores(raw[offScores : offScores+lenScores])
	return d, nil
}

// pages обрезает звук до целого числа 256-байтных страниц — ровно столько
// воспроизводил плеер оригинала.
func pages(b []byte) []byte {
	return b[:len(b)/256*256]
}

// decodePCX распаковывает PCX 640x350 с 4 битовыми плоскостями в индексы цветов.
func decodePCX(b []byte) ([]byte, error) {
	if len(b) < 128 || b[0] != 0x0A {
		return nil, errors.New("титульная картинка: неверный заголовок PCX")
	}
	planes := int(b[0x41])
	bpl := int(binary.LittleEndian.Uint16(b[0x42:]))
	w := int(binary.LittleEndian.Uint16(b[8:])) - int(binary.LittleEndian.Uint16(b[4:])) + 1
	h := int(binary.LittleEndian.Uint16(b[10:])) - int(binary.LittleEndian.Uint16(b[6:])) + 1
	if planes != 4 || w != 640 || h != 350 {
		return nil, fmt.Errorf("титульная картинка: неподдерживаемый формат %dx%d, плоскостей %d", w, h, planes)
	}
	// RLE: байт с двумя старшими единицами — счётчик повтора следующего байта
	raw := make([]byte, 0, bpl*planes*h)
	for p := 128; p < len(b) && len(raw) < cap(raw); {
		c := b[p]
		p++
		if c&0xC0 == 0xC0 {
			n := int(c & 0x3F)
			if p >= len(b) {
				break
			}
			v := b[p]
			p++
			for i := 0; i < n; i++ {
				raw = append(raw, v)
			}
		} else {
			raw = append(raw, c)
		}
	}
	if len(raw) < bpl*planes*h {
		return nil, errors.New("титульная картинка: данные PCX обрываются")
	}
	pix := make([]byte, w*h)
	for y := 0; y < h; y++ {
		row := raw[y*bpl*planes:]
		for x := 0; x < w; x++ {
			bit := 7 - uint(x&7)
			var c byte
			for pl := 0; pl < planes; pl++ {
				c |= ((row[pl*bpl+(x>>3)] >> bit) & 1) << pl
			}
			pix[y*w+x] = c
		}
	}
	return pix, nil
}

// decodeSprite разбирает спрайт собственного формата игры:
// заголовок (ширина-1, высота-1, число плоскостей), затем плоскости
// в порядке 3, 2, 1, 0 и маска; каждая плоскость — h строк по ((w-1)>>3)+1 байт.
func decodeSprite(b []byte) (*Sprite, error) {
	if len(b) < 6 {
		return nil, errors.New("короткий заголовок")
	}
	w := int(binary.LittleEndian.Uint16(b[0:])) + 1
	h := int(binary.LittleEndian.Uint16(b[2:])) + 1
	np := int(binary.LittleEndian.Uint16(b[4:]))
	bpr := ((w - 1) >> 3) + 1
	if np != 4 && np != 5 {
		return nil, fmt.Errorf("неожиданное число плоскостей %d", np)
	}
	if 6+np*h*bpr > len(b) {
		return nil, errors.New("данные обрываются")
	}
	s := &Sprite{W: w, H: h, Pix: make([]byte, w*h), Opaque: make([]bool, w*h)}
	order := [4]int{3, 2, 1, 0}
	plane := func(k, x, y int) bool {
		return (b[6+k*h*bpr+y*bpr+(x>>3)]>>(7-uint(x&7)))&1 != 0
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var c byte
			for k, pl := range order {
				if plane(k, x, y) {
					c |= 1 << pl
				}
			}
			s.Pix[y*w+x] = c
			s.Opaque[y*w+x] = np == 4 || plane(4, x, y)
		}
	}
	return s, nil
}

// ParseHiScores разбирает 130-байтную таблицу рекордов.
func ParseHiScores(b []byte) [10]HiScore {
	var hs [10]HiScore
	for i := range hs {
		rec := b[i*13:]
		hs[i].Score = int32(binary.LittleEndian.Uint32(rec))
		copy(hs[i].Name[:], rec[4:13])
	}
	return hs
}

// EncodeHiScores упаковывает таблицу рекордов в исходный 130-байтный формат.
func EncodeHiScores(hs [10]HiScore) []byte {
	b := make([]byte, lenScores)
	for i, h := range hs {
		rec := b[i*13:]
		binary.LittleEndian.PutUint32(rec, uint32(h.Score))
		copy(rec[4:13], h.Name[:])
	}
	return b
}

// Size возвращает размеры спрайта.
func (s *Sprite) Size() (int, int) { return s.W, s.H }

// At возвращает цвет пикселя и признак непрозрачности.
func (s *Sprite) At(x, y int) (byte, bool) {
	i := y*s.W + x
	return s.Pix[i], s.Opaque[i]
}
