package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/redis/go-redis/v9"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/ai"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/db"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/queue"
)

func findProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	dir := filepath.Dir(filename)

	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}

func main() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		panic("failed to find project root: " + err.Error())
	}

	modelPath := filepath.Join(projectRoot, "infra", "Qwen3VL-4B-Instruct-Q8_0.gguf")
	grammarPath := filepath.Join(projectRoot, "docs", "taxonomy.gbnf")

	// 1. Initialize DB
	dbConn, err := db.NewConnection("postgres://postgres:postgres@localhost:5432/image2taxonomy")
	if err != nil {
		panic(err)
	}

	// 2. Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 3. Initialize AI
	aiEngine, err := ai.NewEngine(modelPath, grammarPath)
	if err != nil {
		panic(err)
	}
	defer aiEngine.Close()

	// 4. Start Blocking Worker
	queue.StartWorker(rdb, aiEngine, dbConn)
}
