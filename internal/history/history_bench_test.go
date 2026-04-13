package history

import (
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkHistoryLoad(b *testing.B) {
	dir := b.TempDir()
	histPath := filepath.Join(dir, "history.gob")

	// Create history with 500 entries
	h := New(histPath)
	for i := 0; i < 500; i++ {
		h.RecordSelectionWithQuery("query"+string(rune('A'+i%26)), "project/path/"+string(rune('A'+i%26)))
	}
	if err := h.Save(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h2 := New(histPath)
		errCh := h2.LoadAsync()
		if err := <-errCh; err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllScoresForQuery(b *testing.B) {
	h := New("")
	now := time.Now()
	for i := 0; i < 1000; i++ {
		path := "group/project/" + string(rune('A'+i%26)) + string(rune('0'+i%10))
		h.selections[path] = SelectionInfo{Timestamps: []time.Time{now.Add(-time.Duration(i) * time.Hour)}}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.GetAllScoresForQuery("test")
	}
}
