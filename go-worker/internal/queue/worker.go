package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/ai"
	"github.com/rivanjarjes/image2taxonomy/worker/internal/db"
)

type SidekiqJob struct {
	Class string        `json:"class"`
	Args  []interface{} `json:"args"`
	JID   string        `json:"jid"`
}

func StartWorker(rdb *redis.Client, aiEngine *ai.Engine, dbConn *db.Postgres) {
	ctx := context.Background()
	queueName := "queue:default"

	fmt.Println("Go Worker Listening on " + queueName)

	for {
		result, err := rdb.BLPop(ctx, 0, queueName).Result()
		if err != nil {
			log.Println("Redis error:", err)
			time.Sleep(1 * time.Second)
			continue
		}

		processJob(result[1], aiEngine, dbConn)
	}
}

func processJob(payload string, aiEngine *ai.Engine, dbConn *db.Postgres) {
	var job SidekiqJob
	json.Unmarshal([]byte(payload), &job)

	if job.Class != "ProductAnalysisJob" {
		return
	}

	if len(job.Args) < 2 {
		log.Printf("Invalid job args: expected 2, got %d\n", len(job.Args))
		return
	}

	productID := int(job.Args[0].(float64))
	imagePath := job.Args[1].(string)

	fmt.Printf("Processing Product ID: %d | Image: %s\n", productID, imagePath)

	jsonResult, err := aiEngine.AnalyzeImage(imagePath)
	if err != nil {
		log.Println("AI Failure:", err)
		errorJSON := fmt.Sprintf(`{"error_message": %q}`, err.Error())
		dbConn.UpdateStatus(productID, "failed", errorJSON)
		return
	}

	fmt.Printf("AI Result (raw): %s\n", jsonResult)

	// Clean and validate JSON
	cleanedJSON, err := cleanJSON(jsonResult)
	if err != nil {
		log.Printf("JSON cleaning failed: %v\n", err)
		errorJSON := fmt.Sprintf(`{"error_message": %q}`, fmt.Sprintf("JSON parsing error: %v", err))
		dbConn.UpdateStatus(productID, "failed", errorJSON)
		return
	}

	fmt.Printf("AI Result (cleaned): %s\n", cleanedJSON)

	err = dbConn.UpdateStatus(productID, "complete", cleanedJSON)
	if err != nil {
		log.Printf("DB Update Failed: %v\n", err)
		return
	}

	fmt.Println("Success! Updated DB.")
}

func cleanJSON(input string) (string, error) {
	// Remove excessive whitespace
	input = strings.TrimSpace(input)

	// Parse JSON to validate and reformat
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Re-marshal to compact JSON
	compact, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(compact), nil
}
