#!/bin/bash
REGISTRY=${REGISTRY:-"docker.io"}
IMAGE=${IMAGE:-"jkremser/log2rbac"}

prelude() {
    export COSIGN_KEY=$(mktemp /tmp/cosign.XXXXXXXXXX) || exit 1
    export COSIGN_PUB=$(mktemp /tmp/cosign.XXXXXXXXXX) || exit 1
    export FULL_IMG="${REGISTRY}"/"${IMAGE}"@"${containerdigest}"

    # COSGIN_PASSWORD should be passed as environment variable
    echo "${COSIGN_PRIVATE_KEY}" > "$COSIGN_KEY"
    echo "${COSIGN_PUBLIC_KEY}" > "$COSIGN_PUB"
}

main() {
    [[ "${CI}" != "true" ]] && echo "CI not detected, exiting" && exit 1
    [[ $# -lt 1 ]] && echo "usage: ksigh-image.sh <tag>]" && exit 1
    TAG=${1}
    fullcontainerdigest=$(docker inspect "${REGISTRY}"/"${IMAGE}":"${TAG}" --format '{{ index .RepoDigests 0 }}')
    [[ $? != "0" ]] && echo "Can't get the container digest" && exit 1
    containerdigest=$(echo ${fullcontainerdigest} | cut -d '@' -f 2)
    [[ "${DIGEST_ONLY}" == "true" ]] && echo "${containerdigest}" && exit 0

    prelude()

    [[ "${PROVENANCE}" == "true" ]] && {
        # attach provenance
        echo "### SLSA Provenance" >> "$GITHUB_STEP_SUMMARY"
        jq .predicate < provenance.json > provenance-predicate.json
        cosign attest --predicate provenance-predicate.json --key "$COSIGN_KEY" --type slsaprovenance "${FULL_IMG}"
        {
            echo "SLSA Provenance file is attested. You can verify it with the following command."
            echo '```bash'
            echo "cosign verify-attestation --key cosign.pub --type slsaprovenance ${FULL_IMG} | jq '.payload |= @base64d | .payload | fromjson | select(.predicateType==\"https://slsa.dev/provenance/v0.2\" ) | .'"
            echo '```'
        } >> "$GITHUB_STEP_SUMMARY"
    } || {
        # sign and verify
        echo '### Sign image' >> "$GITHUB_STEP_SUMMARY"
        cosign sign --key ${FULL_IMG}
        cosign verify --key ${FULL_IMG}
        {
            echo 'Image is signed. You can verify it with the following command:'
            echo '```bash'
            echo "cosign verify --key cosign.pub ${FULL_IMG}"
            echo '```'
        } >> "$GITHUB_STEP_SUMMARY"
    }
}

main $@
