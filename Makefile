setup-ai:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. -DLLAMA_METAL=ON && cmake --build . --config Release
	mkdir -p infra
	cp llama.cpp/build/bin/llama-server infra/llama-server
	# Copy Metal libraries if they exist
	if [ -d "llama.cpp/build/bin" ]; then \
		cp llama.cpp/build/bin/*.dylib infra/ 2>/dev/null || true; \
	fi
	rm -rf llama.cpp
	@echo "AI Server compiled and installed to infra/llama-server"

setup-ai-no-metal:
	git clone https://github.com/ggml-org/llama.cpp
	cd llama.cpp && mkdir build && cd build && cmake .. && cmake --build . --config Release
	mkdir -p infra
	cp llama.cpp/build/bin/llama-server infra/llama-server
	rm -rf llama.cpp
	@echo "AI Server compiled (CPU only) and installed to infra/llama-server"
