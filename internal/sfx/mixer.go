// Package sfx синтезирует звук оригинальной игры: PC-спикер (меандр заданной
// частоты) и оцифрованные сэмплы, которые оригинал проигрывал через спикер
// методом ШИМ.
//
// События задаются во «времени игры» (секунды от старта). Микшер переводит
// его в номер отсчёта с постоянной задержкой, поэтому звук и картинка
// остаются синхронными независимо от размера аудиобуфера.
package sfx

import (
	"encoding/binary"
	"math"
	"sort"
	"sync"
)

// Rate — частота дискретизации вывода.
const Rate = 44100

// pitClock — тактовая частота таймера PC, по которой sound() вычислял делитель.
const pitClock = 1193180

type toneEvent struct {
	at   int64
	freq float64
}

type voice struct {
	at   int64
	data []float32
}

// Mixer — источник аудиопотока (float32, стерео) для Ebiten.
type Mixer struct {
	mu      sync.Mutex
	pos     int64
	latency int64
	volume  float32

	shift   int64 // поправка между временем игры и позицией аудиопотока
	offline bool  // рендер без реального вывода (время игры = время звука)

	tones []toneEvent
	freq  float64
	phase float64

	voices []voice

	lp, hpIn, hpOut float32
}

// NewMixer создаёт микшер с задержкой планирования latency секунд.
func NewMixer(latency float64) *Mixer {
	return &Mixer{latency: int64(latency * Rate), volume: 0.35}
}

func (m *Mixer) frame(t float64) int64 {
	return int64(t*Rate) + m.latency + m.shift
}

// NewOfflineMixer создаёт микшер для рендера в файл: время игры совпадает
// с временем звука, синхронизация не нужна.
func NewOfflineMixer() *Mixer {
	return &Mixer{volume: 0.35, offline: true}
}

// Sync выравнивает время игры t с позицией аудиопотока: если поток
// останавливался (окно свёрнуто) или отстал, события не копятся с задержкой.
func (m *Mixer) Sync(t float64) {
	if m.offline {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.frame(t)
	want := m.pos + m.latency
	if target < m.pos || target > want+m.latency*3 {
		m.shift += want - target
	}
}

// SetVolume задаёт громкость 0..1.
func (m *Mixer) SetVolume(v float64) {
	m.mu.Lock()
	m.volume = float32(v)
	m.mu.Unlock()
}

// Tone включает спикер на частоте freq (Гц) в момент t; freq<=0 — выключить.
// Частота округляется так же, как делитель таймера в sound() Turbo C.
func (m *Mixer) Tone(t, freq float64) {
	f := 0.0
	if freq > 18 {
		div := math.Floor(pitClock / freq)
		if div >= 1 {
			f = pitClock / div
		}
	}
	m.mu.Lock()
	ev := toneEvent{at: m.frame(t), freq: f}
	n := len(m.tones)
	if n > 0 && m.tones[n-1].at > ev.at {
		// события обычно приходят по порядку; на всякий случай поддерживаем сортировку
		i := sort.Search(n, func(i int) bool { return m.tones[i].at > ev.at })
		m.tones = append(m.tones, toneEvent{})
		copy(m.tones[i+1:], m.tones[i:])
		m.tones[i] = ev
	} else {
		m.tones = append(m.tones, ev)
	}
	m.mu.Unlock()
}

// Sample запускает оцифрованный звук (значения ШИМ 1..64 с частоте srcRate)
// в момент t и возвращает его длительность в секундах.
func (m *Mixer) Sample(t float64, pwm []byte, srcRate float64) float64 {
	data := Resample(pwm, srcRate)
	m.mu.Lock()
	m.voices = append(m.voices, voice{at: m.frame(t), data: data})
	m.mu.Unlock()
	return float64(len(pwm)) / srcRate
}

// StopSamples обрывает все звуки, начиная с момента t.
func (m *Mixer) StopSamples(t float64) {
	m.mu.Lock()
	at := m.frame(t)
	for i := range m.voices {
		v := &m.voices[i]
		if end := at - v.at; end < int64(len(v.data)) {
			if end < 0 {
				end = 0
			}
			v.data = v.data[:end]
		}
	}
	m.mu.Unlock()
}

// Resample переводит ШИМ-значения в float32 с частотой Rate (линейная интерполяция).
func Resample(pwm []byte, srcRate float64) []float32 {
	if len(pwm) == 0 {
		return nil
	}
	n := int(float64(len(pwm)) * Rate / srcRate)
	out := make([]float32, n)
	step := srcRate / Rate
	for i := range out {
		p := float64(i) * step
		k := int(p)
		fr := float32(p - float64(k))
		a := (float32(pwm[k]) - 32.5) / 32
		b := a
		if k+1 < len(pwm) {
			b = (float32(pwm[k+1]) - 32.5) / 32
		}
		out[i] = a + (b-a)*fr
	}
	return out
}

// polyBLEP — поправка для подавления наложения спектра у меандра.
func polyBLEP(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

// Render заполняет out монофоническими отсчётами (используется и для тестов).
func (m *Mixer) Render(out []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range out {
		for len(m.tones) > 0 && m.tones[0].at <= m.pos {
			m.freq = m.tones[0].freq
			m.tones = m.tones[1:]
		}
		var v float64
		if m.freq > 0 && m.freq < 20000 {
			dt := m.freq / Rate
			m.phase += dt
			if m.phase >= 1 {
				m.phase -= 1
			}
			sq := 1.0
			if m.phase >= 0.5 {
				sq = -1
			}
			sq += polyBLEP(m.phase, dt)
			sq -= polyBLEP(math.Mod(m.phase+0.5, 1), dt)
			v = sq * 0.6
		}
		var pcm float32
		alive := m.voices[:0]
		for _, vc := range m.voices {
			idx := m.pos - vc.at
			if idx < 0 {
				alive = append(alive, vc)
				continue
			}
			if idx < int64(len(vc.data)) {
				pcm += vc.data[idx]
				alive = append(alive, vc)
			}
		}
		m.voices = alive
		x := float32(v) + pcm
		// маленький динамик: срез низких (~60 Гц) и мягкий срез высоких (~7 кГц)
		hp := 0.991 * (m.hpOut + x - m.hpIn)
		m.hpIn, m.hpOut = x, hp
		m.lp += 0.62 * (hp - m.lp)
		out[i] = m.lp * m.volume
		m.pos++
	}
}

// Read реализует io.Reader: стерео float32 little-endian.
func (m *Mixer) Read(p []byte) (int, error) {
	frames := len(p) / 8
	if frames == 0 {
		return 0, nil
	}
	buf := make([]float32, frames)
	m.Render(buf)
	for i, v := range buf {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(p[i*8:], bits)
		binary.LittleEndian.PutUint32(p[i*8+4:], bits)
	}
	return frames * 8, nil
}
