package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/redis/go-redis/v9"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/ai"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/db"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/queue"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Model        string `yaml:"model"`
	Taxonomy     string `yaml:"taxonomy"`
	Database     string `yaml:"database"`
	Acceleration string `yaml:"acceleration"` // metal, gpu, cpu, arm
	GPULayers    int    `yaml:"gpu_layers"`   // Number of layers to offload to GPU/Metal
}

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

func loadConfig(projectRoot string) (*Config, error) {
	configPath := filepath.Join(projectRoot, "config.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		panic("failed to find project root: " + err.Error())
	}

	// Load configuration from config.yml
	config, err := loadConfig(projectRoot)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	llamaServerPath := filepath.Join(projectRoot, "infra", "llama", "llama-server")
	modelPath := filepath.Join(projectRoot, "infra", "models", config.Model)
	grammarPath := filepath.Join(projectRoot, config.Taxonomy)

	// 1. Initialize DB
	dbConn, err := db.NewConnection(config.Database)
	if err != nil {
		panic(err)
	}

	// 2. Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 3. Initialize AI
	aiEngine, err := ai.NewEngine(llamaServerPath, modelPath, grammarPath, config.Acceleration, config.GPULayers)
	if err != nil {
		panic(err)
	}
	defer aiEngine.Close()

	// 4. Start Blocking Worker
	queue.StartWorker(rdb, aiEngine, dbConn)
}
