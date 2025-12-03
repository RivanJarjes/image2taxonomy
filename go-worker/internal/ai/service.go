package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rivanjarjes/image2taxonomy/worker/internal/image"
)

type Engine struct {
	cmd     *exec.Cmd
	apiURL  string
	grammar string
}

func NewEngine(llamaServerPath string, modelPath string, grammarPath string, acceleration string, gpuLayers int) (*Engine, error) {
	grammarBytes, err := os.ReadFile(grammarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read grammar: %w", err)
	}

	// Detect mmproj file - it should be in the same directory with "mmproj-" prefix
	modelDir := filepath.Dir(modelPath)
	modelBaseName := filepath.Base(modelPath)
	mmprojPath := filepath.Join(modelDir, "mmproj-"+modelBaseName)

	// Build command arguments
	args := []string{
		"-m", modelPath,
		"--port", "8080",
		"-c", "8192", // Increased for high-resolution product images (was 2048)
	}

	// Add acceleration-specific flags
	switch acceleration {
	case "metal":
		fmt.Printf("Using Metal acceleration (Apple Silicon) with %d GPU layers\n", gpuLayers)
		args = append(args, "-ngl", fmt.Sprintf("%d", gpuLayers))
		args = append(args, "--device", "metal")
	case "gpu":
		fmt.Printf("Using GPU acceleration (CUDA) with %d GPU layers\n", gpuLayers)
		args = append(args, "-ngl", fmt.Sprintf("%d", gpuLayers))
		args = append(args, "--device", "cuda")
	case "arm":
		fmt.Println("Using ARM NEON acceleration")
		args = append(args, "-ngl", fmt.Sprintf("%d", gpuLayers))
		// ARM NEON is typically auto-detected, but we can set it explicitly
		args = append(args, "--device", "arm")
	case "cpu":
		fmt.Println("Using CPU-only mode (no GPU acceleration)")
		// Don't add -ngl for CPU mode
	default:
		fmt.Printf("Unknown acceleration backend '%s', falling back to CPU mode\n", acceleration)
		// Fallback to CPU mode
	}

	// Add mmproj if it exists (required for vision models)
	if _, err := os.Stat(mmprojPath); err == nil {
		fmt.Printf("Found multimodal projector: %s\n", mmprojPath)
		args = append(args, "--mmproj", mmprojPath)
	} else {
		fmt.Printf("Warning: No mmproj file found at %s\n", mmprojPath)
		fmt.Println("Vision capabilities may not work without mmproj file")
	}

	cmd := exec.Command(llamaServerPath, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set DYLD_LIBRARY_PATH to point to infra directory where dylib files are located
	llamaBasePath := filepath.Dir(llamaServerPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DYLD_LIBRARY_PATH=%s", llamaBasePath))

	fmt.Println("Starting llama server...")
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start llama server: %w", err)
	}

	if !waitForServer("http://localhost:8080/health", 2*time.Minute) {
		cmd.Process.Kill()
		return nil, fmt.Errorf("llama-server failed to start after 2 minutes")
	}

	return &Engine{
		cmd:     cmd,
		apiURL:  "http://localhost:8080",
		grammar: string(grammarBytes),
	}, nil
}

func (e *Engine) Close() {
	if e.cmd != nil && e.cmd.Process != nil {
		fmt.Println("Stopping llama server...")
		e.cmd.Process.Kill()
	}
}

func (e *Engine) AnalyzeImage(imagePath string) (string, error) {
	// Resize image to 768px on shortest side for optimal LLM processing
	resizedPath, err := image.ResizeToMinDimension(imagePath, 768)
	if err != nil {
		return "", fmt.Errorf("failed to resize image: %w", err)
	}

	// Clean up temp file if a new one was created
	if resizedPath != imagePath {
		defer os.Remove(resizedPath)
	}

	base64Image, err := encodeFileToBase64(resizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	systemPrompt := `You are a product analysis assistant for fashion images.
	You must:
	- Identify the single main product being sold in the image (the primary focus).
	- Classify it using the provided apparel taxonomy.
	- Always select the MOST SPECIFIC leaf category that applies (never stop at a broad node like "Apparel & Accessories").
	
	For example, if the main visible product is:
	- Any clothing on the upper body will be classified under "Clothing > Clothing Tops"
	- A full suit or tuxedo: use a "Suits > Tuxedos" style path.
	- A jacket or blazer: use the appropriate "Outerwear" / "Coats & Jackets" path.
	- DO NOT use Skirt Suits or Pant Suits taxonomy if the product is not a feminine skirt suit or feminine pant suit. If it isn't a tuxedo, use Clothing > Suits instead!
	- Footwear: only use "Shoes" branches (e.g. Boots, Sneakers) when the product is clearly footwear.
	- Costumes: only use "Costumes" branches (e.g. Halloween, Mardi Gras, etc.) when the product is clearly a costume.
	- Any handbag, wallet, backpack, case, etc.: Only use "Handbags, Wallets & Cases" branches when the product is clearly a handbag, wallet, backpack, case, etc.
	- For hangbags, pay attention to the type of bag, such as crossbody, shoulder, tote, backpack, etc.
	- Landyards, Keychains, Wallet Chains: Only use "Handbag & Wallet Accessories" branches when the product is clearly a handbag accessory or wallet accessory.
	- Belts, Hats, Wristbands, etc.: Only use "Clothing Accessories" branches when the product is clearly a belt, hat, wristband, etc.
	- Any other wearables that don't fit under regular clothing or accessories or bags: Use "Clothing Accessories" branches when the product is clearly a wearable that doesn't fit under regular clothing, accessories, or bags.
	- All jewelry, such as chains, bracelets, earrings, rings, chain belts, etc.: Only use "Jewelry" branches when the product is clearly a piece of jewelry.
	- All jewelry on the body, such as chain belts, nose rings, belly rings, toe rings, etc.: Use "Jewelry > Body jewelry" branches when the product is clearly a piece of jewelry on the body.
	- Watches: Use "Jewelry > Watches" branches when the product is clearly a watch.
	- Smartwatches: Use "Jewelry > Smartwatches" branches when the product is clearly a smartwatch.
	- Anything shoe related but aren't shoes, such as shoe covers, grips, gel pads, shoelaces, etc.: Use "Shoe Accessories" branches when the product is clearly a shoe accessory.
	- Any bras, bodysuits, jockstraps, other lingerie items: Use "Lingerie" branches when the product is clearly a lingerie item.

	The taxonomy string MUST be a valid path starting with "Apparel & Accessories" and using only exact names from the taxonomy.`

	userPrompt := `Analyze the product in this image and provide a JSON response.

	1. TITLE: A specific, descriptive product name for the main item being sold.
	2. DESCRIPTION: Describe the real visual details - colors, materials, design features, branding, style, and what garment or footwear it is.
	3. TAXONOMY: Map the main product to the most specific valid category path from the apparel taxonomy.
	
	Rules for TAXONOMY:
	- Always start with: "Apparel & Accessories > ..."
	- Only use category names that exist in the taxonomy.
	- Always go to the most specific leaf possible.
	- Do NOT output just "Apparel & Accessories".
	
	Respond with JSON only (no other text).`

	payload := map[string]interface{}{
		"model": "qwen3vl",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": userPrompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
				},
			},
		},
		"max_tokens":  768,
		"temperature": 0.05,
		"grammar":     e.grammar,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Printf("Sending chat request (payload size: %d bytes)\n", len(jsonData))

	req, err := http.NewRequest("POST", e.apiURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("Making request to: %s\n", e.apiURL+"/v1/chat/completions")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse OpenAI-compatible response format
	var chatResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &chatResponse); err != nil {
		return "", fmt.Errorf("failed to parse chat response: %w\nResponse: %s", err, string(bodyBytes))
	}

	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices in response: %s", string(bodyBytes))
	}

	// Check if generation was truncated
	if chatResponse.Choices[0].FinishReason == "length" {
		fmt.Println("Warning: AI generation was truncated (hit token limit)")
	}

	content := chatResponse.Choices[0].Message.Content
	fmt.Printf("AI generated %d characters, finish_reason: %s\n",
		len(content), chatResponse.Choices[0].FinishReason)

	return content, nil
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				// Accept 200 (ready) or 503 (loading) - both indicate server is running
				if resp.StatusCode == 200 {
					fmt.Println("llama-server is ready!")
					return true
				}
				if resp.StatusCode == 503 {
					fmt.Println("llama-server is loading model...")
				}
			}
		case <-time.After(timeout):
			return false
		}

		if time.Now().After(deadline) {
			return false
		}
	}
}

func encodeFileToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
