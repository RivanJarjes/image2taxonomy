package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/ai"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/db"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/queue"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Model             string `yaml:"model"`
	Taxonomy          string `yaml:"taxonomy"`
	Database          string `yaml:"database"`
	LocalAcceleration string `yaml:"local_acceleration"`  // metal, gpu, cpu, arm
	LocalGPULayers    int    `yaml:"local_gpu_layers"`    // GPU layers for local
	DockerAcceleration string `yaml:"docker_acceleration"` // metal, gpu, cpu, arm
	DockerGPULayers    int    `yaml:"docker_gpu_layers"`   // GPU layers for Docker
}

func findProjectRoot() (string, error) {
	// First, check if we're running in Docker by looking for the mounted config file
	dockerRoot := "/app"
	if _, err := os.Stat(filepath.Join(dockerRoot, "config.yml")); err == nil {
		return dockerRoot, nil
	}

	// Fall back to finding the project root from source code location (local development)
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

func isRunningInDocker() bool {
	// Check for /.dockerenv file (created by Docker)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// Check for GOLANG_ENV=production (set in docker-compose.yml)
	if os.Getenv("GOLANG_ENV") == "production" {
		return true
	}
	// Check if running from /app (Docker working directory)
	if cwd, err := os.Getwd(); err == nil && strings.HasPrefix(cwd, "/app") {
		return true
	}
	return false
}

func main() {
	// Parse command-line flags
	dockerMode := flag.Bool("docker", false, "Run in Docker mode (uses docker_* config settings)")
	localMode := flag.Bool("local", false, "Run in local mode (uses local_* config settings)")
	flag.Parse()

	projectRoot, err := findProjectRoot()
	if err != nil {
		panic("failed to find project root: " + err.Error())
	}

	// Load configuration from config.yml
	config, err := loadConfig(projectRoot)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Determine which mode to use: flag > auto-detect
	var useDockerSettings bool
	if *dockerMode {
		useDockerSettings = true
		fmt.Println("Running in Docker mode (--docker flag)")
	} else if *localMode {
		useDockerSettings = false
		fmt.Println("Running in local mode (--local flag)")
	} else {
		// Auto-detect based on environment
		useDockerSettings = isRunningInDocker()
		if useDockerSettings {
			fmt.Println("Auto-detected Docker environment")
		} else {
			fmt.Println("Auto-detected local environment")
		}
	}

	// Select acceleration settings based on mode
	var acceleration string
	var gpuLayers int
	if useDockerSettings {
		acceleration = config.DockerAcceleration
		gpuLayers = config.DockerGPULayers
		fmt.Printf("Using Docker settings: acceleration=%s, gpu_layers=%d\n", acceleration, gpuLayers)
	} else {
		acceleration = config.LocalAcceleration
		gpuLayers = config.LocalGPULayers
		fmt.Printf("Using local settings: acceleration=%s, gpu_layers=%d\n", acceleration, gpuLayers)
	}

	llamaServerPath := filepath.Join(projectRoot, "infra", "llama", "llama-server")
	modelPath := filepath.Join(projectRoot, "infra", "models", config.Model)
	grammarPath := filepath.Join(projectRoot, config.Taxonomy)

	// 1. Initialize DB - prefer DATABASE_URL env var, fall back to config.yml
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = config.Database
	}
	dbConn, err := db.NewConnection(databaseURL)
	if err != nil {
		panic(err)
	}

	// 2. Initialize Redis - prefer REDIS_URL env var, fall back to localhost
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	// Parse Redis URL using the redis library's built-in parser
	rdbOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback: parse manually if ParseURL fails
		parsedURL, parseErr := url.Parse(redisURL)
		if parseErr != nil || parsedURL.Host == "" {
			// Last resort: use localhost
			rdbOpts = &redis.Options{
				Addr: "localhost:6379",
			}
		} else {
			redisDB := 0
			if parsedURL.Path != "" {
				dbStr := strings.TrimPrefix(parsedURL.Path, "/")
				if dbStr != "" {
					// Try to parse DB number from path
					if dbNum, parseDBErr := strconv.Atoi(dbStr); parseDBErr == nil {
						redisDB = dbNum
					}
				}
			}
			rdbOpts = &redis.Options{
				Addr: parsedURL.Host,
				DB:   redisDB,
			}
		}
	}

	rdb := redis.NewClient(rdbOpts)

	// 3. Initialize AI
	aiEngine, err := ai.NewEngine(llamaServerPath, modelPath, grammarPath, acceleration, gpuLayers)
	if err != nil {
		panic(err)
	}
	defer aiEngine.Close()

	// 4. Start Blocking Worker
	queue.StartWorker(rdb, aiEngine, dbConn)
}
