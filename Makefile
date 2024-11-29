IMAGE_VERSION = latest
REGISTRY = docker.io/kaynwong
IMAGE = ${REGISTRY}/custom-device-plugin:${IMAGE_VERSION}

build:
	docker buildx build --platform linux/amd64,linux/arm64 -t ${IMAGE} . --push

kindLoad:
	docker pull ${IMAGE} && kind load docker-image ${IMAGE}