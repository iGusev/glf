package index

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkNewDescriptionIndex(b *testing.B) {
	dir := b.TempDir()
	indexPath := filepath.Join(dir, "bench.bleve")

	// Create index once
	idx, err := NewDescriptionIndex(indexPath)
	if err != nil {
		b.Fatal(err)
	}
	idx.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx, err := NewDescriptionIndex(indexPath)
		if err != nil {
			b.Fatal(err)
		}
		idx.Close()
	}
}

func BenchmarkSearch(b *testing.B) {
	dir := b.TempDir()
	indexPath := filepath.Join(dir, "bench.bleve")

	idx, err := NewDescriptionIndex(indexPath)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()

	docs := make([]DescriptionDocument, 500)
	for i := range docs {
		docs[i] = DescriptionDocument{
			ProjectPath: filepath.Join("group", "subgroup", "project"+string(rune('A'+i%26))+string(rune('0'+i%10))),
			ProjectName: "Project " + string(rune('A'+i%26)),
			Description: "A test project for benchmarking search performance",
			Member:      true,
		}
	}
	if err := idx.AddBatch(docs); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := idx.Search("project", 100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllProjects(b *testing.B) {
	dir := b.TempDir()
	indexPath := filepath.Join(dir, "bench.bleve")

	idx, err := NewDescriptionIndex(indexPath)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()

	docs := make([]DescriptionDocument, 1000)
	for i := range docs {
		docs[i] = DescriptionDocument{
			ProjectPath: filepath.Join("group", "project"+string(rune('A'+i%26))+string(rune('0'+i%10))),
			ProjectName: "Project " + string(rune('A'+i%26)),
			Description: "Benchmark project",
			Member:      true,
		}
	}
	if err := idx.AddBatch(docs); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := idx.GetAllProjects()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddBatch(b *testing.B) {
	docs := make([]DescriptionDocument, 500)
	for i := range docs {
		docs[i] = DescriptionDocument{
			ProjectPath: filepath.Join("group", "project"+string(rune('A'+i%26))+string(rune('0'+i%10))),
			ProjectName: "Project " + string(rune('A'+i%26)),
			Description: "Benchmark project for batch indexing",
			Member:      true,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dir := b.TempDir()
		indexPath := filepath.Join(dir, "bench.bleve")
		idx, err := NewDescriptionIndex(indexPath)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if err := idx.AddBatch(docs); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		idx.Close()
		os.RemoveAll(dir)
		b.StartTimer()
	}
}
