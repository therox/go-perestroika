package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/therox/go-perestroika/internal/dat"
)

// Перетащенный в окно файл проверяется, копируется в каталог игры и
// при следующем запуске находится сам.
func TestInstallDataFile(t *testing.T) {
	path := os.Getenv("PERESTROIKA_DAT")
	if path == "" {
		path = "../../Source/TOPPLER.DAT"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skip("нет файла данных игры (задайте PERESTROIKA_DAT):", err)
	}
	dataDirOverride = t.TempDir()
	defer func() { dataDirOverride = "" }()

	if _, err := InstallDataFile([]byte("не тот файл")); !errors.Is(err, dat.ErrFormat) {
		t.Fatalf("посторонний файл: %v", err)
	}
	if _, err := InstallDataFile(raw); err != nil {
		t.Fatal(err)
	}
	if p := findInDir(dataDirOverride); p != filepath.Join(dataDirOverride, DataFileName) {
		t.Fatalf("файл не установлен: %q", p)
	}
	if _, err := LoadDataFile(filepath.Join(dataDirOverride, DataFileName)); err != nil {
		t.Fatal(err)
	}
}
