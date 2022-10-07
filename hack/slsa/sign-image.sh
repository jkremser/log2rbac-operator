#!/bin/bash
[[ "x${DEBUG}" == "xtrue" ]] && set -x

REGISTRIES=${REGISTRIES:-"docker.io ghcr.io"}
IMAGE=${IMAGE:-"jkremser/log2rbac"}

prelude() {
    export COSIGN_KEY=$(mktemp /tmp/cosign.XXXXXXXXXX) || exit 1
    export COSIGN_PUB=$(mktemp /tmp/cosign.XXXXXXXXXX) || exit 1
    export IMG="${IMAGE}"@"${containerdigest}"

    # COSGIN_PASSWORD should be passed as environment variable
    echo "${COSIGN_PRIVATE_KEY}" > "$COSIGN_KEY"
    echo "${COSIGN_PUBLIC_KEY}" > "$COSIGN_PUB"
}

main() {
    [[ "x${CI}" != "xtrue" ]] && echo "CI not detected, exiting" && exit 1
    [[ -z "${GITHUB_STEP_SUMMARY}" ]] && echo "GITHUB_STEP_SUMMARY not set, exiting" && exit 1
    [[ $# -lt 1 ]] && echo "usage: ksigh-image.sh <tag>]" && exit 1
    TAG=${1}
    docker pull "docker.io"/"${IMAGE}":"${TAG}" &> /dev/null
    fullcontainerdigest=$(docker inspect "docker.io/${IMAGE}":"${TAG}" --format '{{ index .RepoDigests 0 }}')
    [[ $? != "0" ]] && echo "Can't get the container digest" && exit 1
    containerdigest=$(echo ${fullcontainerdigest} | cut -d '@' -f 2)
    [[ "x${DIGEST_ONLY}" == "xtrue" ]] && echo "${containerdigest}" && exit 0

    prelude
    for registry in ${REGISTRIES}; do
        FULL_IMG="${registry}/${IMG}"
        [[ "x${PROVENANCE}" == "xtrue" ]] && {
            # attach provenance
            echo "### SLSA Provenance for ${registry}" >> "$GITHUB_STEP_SUMMARY"
            jq .predicate < ${registry}.provenance > ${registry}-provenance-predicate.json
            cosign attest --predicate ${registry}-provenance-predicate.json --key "$COSIGN_KEY" --type slsaprovenance "${FULL_IMG}" || exit 2
            {
                echo "SLSA Provenance file is attested. You can verify it with the following command."
                echo '```bash'
                echo "cosign verify-attestation --key cosign.pub --type slsaprovenance ${FULL_IMG} | jq '.payload |= @base64d | .payload | fromjson | select(.predicateType==\"https://slsa.dev/provenance/v0.2\" ) | .'"
                echo '```'
            } >> "$GITHUB_STEP_SUMMARY"
        } || {
            # sign and verify
            echo "### Sign image for ${registry}" >> "$GITHUB_STEP_SUMMARY"
            cosign sign --key ${COSIGN_KEY} ${FULL_IMG} || exit 2
            cosign verify --key ${COSIGN_PUB} ${FULL_IMG} || exit 2
            {
                echo 'Image is signed. You can verify it with the following command:'
                echo '```bash'
                echo "cosign verify --key cosign.pub ${FULL_IMG}"
                echo '```'
            } >> "$GITHUB_STEP_SUMMARY"
        }
    done
}

main $@
