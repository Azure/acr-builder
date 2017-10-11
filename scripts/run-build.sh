#/bin/bash
set -e

if [[ -z "$ACR_BUILDER_IMAGE" ]]; then
    if [[ ! -z "$ACR_BUILD_DOCKER_REGISTRY" ]]; then
        ACR_BUILDER_IMAGE="${ACR_BUILD_DOCKER_REGISTRY}/acr-builder"
    else
        ACR_BUILDER_IMAGE="acr-builder"
    fi
fi

if [[ "$(docker images -q $ACR_BUILDER_IMAGE 2> /dev/null)" == "" ]]; then
	echo "Expected image $ACR_BUILDER_IMAGE to exist. Please build or pull the image prior to running..." 1>&2
	exit 1
fi

docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v `pwd`:/root/project -w /root/project $ACR_BUILDER_IMAGE "$@"
