package main

import (
	"testing"
	"time"

	"github.com/therox/go-perestroika/internal/gfx"
	"github.com/therox/go-perestroika/internal/host"
)

// Картинка показывается с заданной задержкой: на экран попадает кадр,
// снятый примерно AudioDelayMs миллисекунд назад.
func TestDelayedFrame(t *testing.T) {
	a := &app{scr: gfx.NewScreen(), settings: host.Settings{AudioDelayMs: 100}}
	paint := func(c int) {
		a.scr.SetFillStyle(gfx.SolidFill, c)
		a.scr.Bar(0, 0, gfx.MaxX, gfx.MaxY)
	}
	var shown []byte
	for c := 1; c <= 12; c++ {
		paint(c)
		shown = append(shown, a.delayedFrame().Pix[0])
		time.Sleep(25 * time.Millisecond)
	}
	// первые ~100 мс показывается самый первый кадр, дальше — отставание на 4 кадра
	if shown[0] != 1 || shown[2] != 1 {
		t.Fatalf("начало: %v", shown)
	}
	if last := shown[len(shown)-1]; last < 7 || last > 9 {
		t.Fatalf("отставание неверное: %v", shown)
	}
	if len(a.history) > 6 {
		t.Fatalf("старые кадры не освобождаются: %d", len(a.history))
	}

	a.settings.AudioDelayMs = 0
	paint(14)
	if got := a.delayedFrame().Pix[0]; got != 14 {
		t.Fatalf("без задержки показан кадр %d", got)
	}
}
