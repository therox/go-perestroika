package host

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/therox/go-perestroika/internal/dat"
)

// DataFileName — имя файла данных игры.
const DataFileName = "TOPPLER.DAT"

// ErrNoData — файл данных не найден.
var ErrNoData = errors.New("не найден файл данных " + DataFileName)

// dataDirOverride подменяет каталог игры (для тестов).
var dataDirOverride string

// DataDir возвращает каталог игры в настройках пользователя
// (там хранятся рекорды и скопированный файл данных).
func DataDir() string {
	if dataDirOverride != "" {
		return dataDirOverride
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, "perestroika")
}

// DefaultHiScorePath возвращает путь к файлу рекордов.
func DefaultHiScorePath() string {
	return filepath.Join(DataDir(), "hiscores.dat")
}

// FindDataFile ищет TOPPLER.DAT: рядом с программой, в текущем каталоге
// и в каталоге игры в настройках пользователя. Регистр имени не важен.
func FindDataFile() (string, error) {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	dirs = append(dirs, DataDir())
	for _, d := range dirs {
		if p := findInDir(d); p != "" {
			return p, nil
		}
	}
	return "", ErrNoData
}

func findInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), DataFileName) {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// LoadDataFile читает и разбирает файл данных.
func LoadDataFile(path string) (*dat.Data, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return dat.Parse(raw)
}

// InstallDataFile проверяет файл данных и сохраняет его копию в каталоге игры,
// чтобы в следующий раз он нашёлся сам.
func InstallDataFile(raw []byte) (*dat.Data, error) {
	d, err := dat.Parse(raw)
	if err != nil {
		return nil, err
	}
	dir := DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	tmp := filepath.Join(dir, DataFileName+".tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, filepath.Join(dir, DataFileName)); err != nil {
		return nil, err
	}
	return d, nil
}
