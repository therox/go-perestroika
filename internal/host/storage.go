package host

import (
	"os"
	"path/filepath"

	"github.com/therox/go-perestroika/internal/dat"
)

// HiScoreFile хранит таблицу рекордов в исходном 130-байтном формате
// в каталоге настроек пользователя (оригинал писал её в конец TOPPLER.DAT).
type HiScoreFile struct {
	Path string
}

// Load читает таблицу; если файла нет — возвращает def.
func (f *HiScoreFile) Load(def [10]dat.HiScore) [10]dat.HiScore {
	b, err := os.ReadFile(f.Path)
	if err != nil || len(b) != 130 {
		return def
	}
	return dat.ParseHiScores(b)
}

// SaveHiScores записывает таблицу.
func (f *HiScoreFile) SaveHiScores(hs [10]dat.HiScore) error {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return err
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, dat.EncodeHiScores(hs), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}
