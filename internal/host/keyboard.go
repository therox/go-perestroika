// Package host — окружение игры: часы, клавиатура, хранилище рекордов.
package host

import (
	"sync"

	"github.com/therox/go-perestroika/internal/dat"
)

// Keyboard — клавиатура в стиле PC: буфер BIOS и очередь скан-кодов для
// собственного обработчика прерывания игры.
type Keyboard struct {
	mu    sync.Mutex
	isr   bool
	keys  []int
	scans []byte
}

// Ёмкость буфера клавиатуры BIOS.
const biosBufSize = 15

// SetISR включает режим собственного обработчика прерывания (во время этапа).
func (k *Keyboard) SetISR(on bool) {
	k.mu.Lock()
	k.isr = on
	k.scans = k.scans[:0]
	k.mu.Unlock()
}

// Press регистрирует нажатие: scan — скан-код, ascii — символ (0 для служебных клавиш).
func (k *Keyboard) Press(scan, ascii byte) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.isr {
		if scan != 0 && len(k.scans) < 16 {
			k.scans = append(k.scans, scan)
		}
		return
	}
	if len(k.keys) < biosBufSize {
		k.keys = append(k.keys, int(scan)<<8|int(ascii))
	}
}

// KeyHit — есть ли клавиша в буфере BIOS.
func (k *Keyboard) KeyHit() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.keys) > 0
}

// ReadKey извлекает клавишу из буфера BIOS.
func (k *Keyboard) ReadKey() (int, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.keys) == 0 {
		return 0, false
	}
	v := k.keys[0]
	k.keys = k.keys[1:]
	return v, true
}

// PopScan извлекает скан-код для обработчика прерывания.
func (k *Keyboard) PopScan() (byte, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.scans) == 0 {
		return 0, false
	}
	v := k.scans[0]
	k.scans = k.scans[1:]
	return v, true
}

// Flush очищает оба буфера.
func (k *Keyboard) Flush() {
	k.mu.Lock()
	k.keys = k.keys[:0]
	k.scans = k.scans[:0]
	k.mu.Unlock()
}

// RuneToCP866 переводит печатный символ в кодировку CP866 (0 — символ не поддерживается).
func RuneToCP866(r rune) byte {
	if r < 0x20 || r == 0x7F {
		return 0
	}
	b, ok := dat.RuneToCP866(r)
	if !ok {
		return 0
	}
	return b
}
