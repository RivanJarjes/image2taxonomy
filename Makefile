# Default setup (auto-detect or use Metal for Apple Silicon)
setup-ai:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. -DLLAMA_METAL=ON && cmake --build . --config Release
	mkdir -p infra/llama
	cp llama.cpp/build/bin/llama-server infra/llama/llama-server
	if [ -d "llama.cpp/build/bin" ]; then \
		cp llama.cpp/build/bin/*.dylib infra/ 2>/dev/null || true; \
	fi
	rm -rf llama.cpp
	@echo "✓ llama-server compiled with Metal acceleration and installed to infra/llama"

# Metal acceleration (Apple Silicon / M1 / M2 / M3 etc)
setup-ai-metal:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. -DLLAMA_METAL=ON && cmake --build . --config Release
	mkdir -p infra/llama
	cp llama.cpp/build/bin/llama-server infra/llama/llama-server
	if [ -d "llama.cpp/build/bin" ]; then \
		cp llama.cpp/build/bin/*.dylib infra/ 2>/dev/null || true; \
	fi
	rm -rf llama.cpp
	@echo "✓ llama-server compiled with Metal acceleration (Apple Silicon)"
	@echo "  Update config.yml: acceleration: metal"

# GPU acceleration (NVIDIA CUDA)
setup-ai-gpu:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. -DLLAMA_CUDA=ON && cmake --build . --config Release
	mkdir -p infra/llama
	cp llama.cpp/build/bin/llama-server infra/llama/llama-server
	rm -rf llama.cpp
	@echo "✓ llama-server compiled with CUDA GPU acceleration (NVIDIA)"
	@echo "  Update config.yml: acceleration: gpu"

# ARM NEON acceleration (ARM processors)
setup-ai-arm:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. -DLLAMA_NEON=ON && cmake --build . --config Release
	mkdir -p infra/llama
	cp llama.cpp/build/bin/llama-server infra/llama/llama-server
	rm -rf llama.cpp
	@echo "✓ llama-server compiled with ARM NEON acceleration"
	@echo "  Update config.yml: acceleration: arm"

# CPU-only (no GPU acceleration)
setup-ai-cpu:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. && cmake --build . --config Release
	mkdir -p infra/llama
	cp llama.cpp/build/bin/llama-server infra/llama/llama-server
	rm -rf llama.cpp
	@echo "✓ llama-server compiled (CPU only - no GPU acceleration)"
	@echo "  Update config.yml: acceleration: cpu"

setup-ai-no-metal: setup-ai-cpu

# Docker targets

# Start all services
docker-up:
	docker-compose up -d
	@echo "✓ All services started"
	@echo "  Rails: http://localhost:3000"
	@echo "  PgAdmin: http://localhost:5050"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis: localhost:6379"

# Stop all services (keep volumes)
docker-down:
	docker-compose down
	@echo "✓ All services stopped"

# View logs from all services
docker-logs:
	docker-compose logs -f

# View logs from specific service (usage: make docker-logs-service SERVICE=go-worker)
docker-logs-service:
	docker-compose logs -f $(SERVICE)

# Build all services
docker-build:
	docker-compose build --no-cache
	@echo "✓ All services built"

# Rebuild Go worker only
docker-build-worker:
	docker-compose build --no-cache go-worker
	@echo "✓ Go worker built"

# Rebuild Rails app only
docker-build-rails:
	docker-compose build --no-cache rails-app
	@echo "✓ Rails app built"

# Check service status
docker-status:
	docker-compose ps

# Run Rails migrations
docker-migrate:
	docker-compose exec rails-app ./bin/rails db:migrate
	@echo "✓ Migrations completed"

# Access PostgreSQL shell
docker-psql:
	docker-compose exec postgres psql -U postgres -d image2taxonomy

# Access Redis CLI
docker-redis:
	docker-compose exec redis redis-cli

# Access Go worker shell
docker-worker-shell:
	docker-compose exec go-worker sh

# Access Rails console
docker-rails-console:
	docker-compose exec rails-app ./bin/rails console

# Clean up everything (removes volumes - CAREFUL!)
docker-clean:
	docker-compose down -v
	@echo "✓ All services and volumes removed"

# View resource usage
docker-stats:
	docker-compose stats

# Restart all services
docker-restart:
	docker-compose restart
	@echo "✓ All services restarted"

# Initialize everything from scratch
docker-init: setup-ai docker-build docker-up docker-migrate
	@echo "✓ Complete Docker setup initialized"
	@echo ""
	@echo "Next steps:"
	@echo "1. Check service status: make docker-status"
	@echo "2. View logs: make docker-logs"
	@echo "3. Access Rails: http://localhost:3000"
	@echo "4. Access PgAdmin: http://localhost:5050 (admin@example.com / admin)"

# Health check
docker-health:
	@echo "Checking service health..."
	@docker-compose exec postgres pg_isready -U postgres && echo "✓ PostgreSQL healthy" || echo "✗ PostgreSQL unhealthy"
	@docker-compose exec redis redis-cli ping > /dev/null && echo "✓ Redis healthy" || echo "✗ Redis unhealthy"
	@docker-compose ps | grep -q "Up" && echo "✓ Services running" || echo "✗ Services not running"

.PHONY: docker-up docker-down docker-logs docker-logs-service docker-build docker-build-worker docker-build-rails \
        docker-status docker-migrate docker-psql docker-redis docker-worker-shell docker-rails-console \
        docker-clean docker-stats docker-restart docker-init docker-health
