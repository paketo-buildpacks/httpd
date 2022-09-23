.PHONY: test

id:
	@echo httpd

test:
	@cd test; \
	./run-test --version $(version) --tarballPath $(tarballPath)

retrieve:
	@cd retrieval; \
	go build -o retrieve; \
	./retrieve --buildpack_toml_path=$(buildpackTomlPath) \
		--output=$(output); \
	rm retrieve
