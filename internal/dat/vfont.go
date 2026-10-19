package dat

import "errors"

// VFont — векторный шрифт игры с кириллицей.
//
// Символы кодируются латиницей по раскладке ЙЦУКЕН: строка «Dthcbz» выводится
// как «Версия». Индекс глифа — код символа минус 0x21.
type VFont struct {
	Widths []int
	Glyphs [][]Stroke
}

// Stroke — один штрих глифа: перемещение пера или линия к точке
// (DX, DY) относительно начала глифа; DY отсчитывается вверх от базовой линии.
type Stroke struct {
	Draw   bool
	DX, DY int
}

// MaxIndex возвращает наибольший индекс символа (код-0x21).
func (f *VFont) MaxIndex() int { return len(f.Widths) - 1 }

// Width возвращает ширину символа с индексом idx.
func (f *VFont) Width(idx int) int {
	if idx < 0 || idx >= len(f.Widths) {
		return 0
	}
	return f.Widths[idx]
}

// Glyph возвращает штрихи символа с индексом idx.
func (f *VFont) Glyph(idx int) []Stroke {
	if idx < 0 || idx >= len(f.Glyphs) {
		return nil
	}
	return f.Glyphs[idx]
}

// parseVFont разбирает шрифт в формате TOPPLER.DAT: байт N (число символов-1),
// N+1 ширина, таблица смещений и штрихи — пары байт (dx, dy) с флагом
// «перо опущено» в старшем бите dy.
func parseVFont(b []byte) (*VFont, error) {
	if len(b) < 2 {
		return nil, errors.New("шрифт: пустой блок")
	}
	n := int(b[0])
	if len(b) < 1+(n+1)*3 {
		return nil, errors.New("шрифт: блок обрывается")
	}
	f := &VFont{Widths: make([]int, n+1), Glyphs: make([][]Stroke, n+1)}
	for idx := 0; idx <= n; idx++ {
		f.Widths[idx] = int(b[1+idx])
		p := idx*2 + n + 1
		lo, hi := int(b[p]), int(b[p+1])
		// в оригинале старший байт умножается на 255, а не на 256
		off := n*3 + hi*255 + lo + 1
		for off+1 < len(b) {
			b1, b2 := b[off], b[off+1]
			off += 2
			if b1 == 0 && b2 == 0 {
				break
			}
			// смещение по y — знаковое: значения больше 8 означают отрицательные
			dy := int(b2 & 0x7F)
			if dy > 8 {
				dy -= 0x80
			}
			f.Glyphs[idx] = append(f.Glyphs[idx], Stroke{Draw: b2&0x80 != 0, DX: int(b1 & 0x7F), DY: dy})
		}
	}
	return f, nil
}
