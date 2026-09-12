// Пакет archive: JSON-архів зниклих/забраних замовлень (orders-YYYY-MM.json).
package archive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

type Archiver struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) *Archiver {
	return &Archiver{dir: dir}
}

// Append додає замовлення у файл поточного місяця,
// створюючи каталог за потреби. Пошкоджений файл не ламає архівацію.
func (a *Archiver) Append(o model.Order) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := os.MkdirAll(a.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(a.dir, fmt.Sprintf("orders-%s.json", time.Now().Format("2006-01")))
	entries := []model.Order{}
	if blob, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(blob, &entries); err != nil {
			entries = []model.Order{}
		}
	}
	entries = append(entries, o)
	blob, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o644)
}
