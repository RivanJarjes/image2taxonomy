# Image2Taxonomy

https://github.com/user-attachments/assets/56c7f036-6961-4d4c-ae0e-16d589d334b2

Image2Taxonomy is a microservice built to automate the e-commerce product classification problem. It uses a Ruby on Rails frontend to manage uploads and orchestrate asynchronous analysis jobs processed by a dedicated Go service running a constrained Multimodal LLM (Qwen-VL) on optimized hardware. The system enforces the official Shopify Product Taxonomy via GBNF Grammars that's automatically updated and generated thru the Go program.

## Prerequisites

- Go
- Ruby
- Docker
- A Qwen3-VL GGUF file (configured to use Qwen3-VL-4B as of right now)
- CMake for Llama-server

## Setup

1. Clone and setup

```bash
git clone https://github.com/rivanjarjes/Image2Taxonomy.git
cd Image2Taxonomy
make setup-ai
```

2. Download you preferred vision language model put it in `infra/models`

3. Run the docker services

```bash
cd infra
docker-compose up -d
cd ..
```

4. Run the Rails application

```bash
cd rails-app
bin/rails db:migrate
bin/dev
```

5. Generate the grammar file and run the Go server:

```bash
cd go-worker
go run ./cmd/gen-grammar/main.go
# wait to finish
go run ./cmd/worker/main.go
```

6. Try out the program at `localhost:3000`
