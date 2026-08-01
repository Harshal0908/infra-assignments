CLUSTER_NAME := config-service
NAMESPACE := config-service
IMAGE_NAME := config-service:latest
TERRAFORM_DIR := infra/terraform

.PHONY: help \
	cluster-create cluster-delete \
	test image-build image-load \
	infra-init infra-apply \
	migrate deploy bootstrap \
	status logs clean \
	validate status logs clean

help:
	@echo "Available commands:"
	@echo "  make test            Run Go tests"
	@echo "  make cluster-create  Create the Kind cluster"
	@echo "  make image-build     Build the Docker image"
	@echo "  make image-load      Load the image into Kind"
	@echo "  make infra-apply     Provision infrastructure"
	@echo "  make migrate         Run the database migration"
	@echo "  make deploy          Deploy the application"
	@echo "  make bootstrap       Build the complete environment"
	@echo "  make status          Show Kubernetes resources"
	@echo "  make logs            Show application logs"
	@echo "  make clean           Delete the local environment"
	@echo "  make validate        Run deployment smoke tests"

test:
	go test ./... -race

cluster-create:
	@if kind get clusters | grep -qx "$(CLUSTER_NAME)"; then \
		echo "Kind cluster $(CLUSTER_NAME) already exists"; \
	else \
		kind create cluster --name "$(CLUSTER_NAME)"; \
	fi

cluster-delete:
	kind delete cluster --name "$(CLUSTER_NAME)"

image-build:
	docker build -t "$(IMAGE_NAME)" .

image-load:
	kind load docker-image "$(IMAGE_NAME)" --name "$(CLUSTER_NAME)"

infra-init:
	terraform -chdir="$(TERRAFORM_DIR)" init

infra-apply:
	terraform -chdir="$(TERRAFORM_DIR)" apply -auto-approve

migrate:
	kubectl apply -f k8s/migration-configmap.yaml
	kubectl -n "$(NAMESPACE)" delete job config-service-migration --ignore-not-found
	kubectl apply -f k8s/migration-job.yaml
	kubectl -n "$(NAMESPACE)" wait \
		--for=condition=complete \
		job/config-service-migration \
		--timeout=120s

deploy:
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl -n "$(NAMESPACE)" rollout status \
		deployment/config-service \
		--timeout=120s

bootstrap: test cluster-create image-build image-load infra-init infra-apply migrate deploy
	@echo "Config service deployment completed successfully."

status:
	kubectl -n "$(NAMESPACE)" get pods,services,pvc,jobs

logs:
	kubectl -n "$(NAMESPACE)" logs deployment/config-service

clean:
	terraform -chdir="$(TERRAFORM_DIR)" destroy -auto-approve || true
	kind delete cluster --name "$(CLUSTER_NAME)"
validate:
	./scripts/smoke-test.sh
