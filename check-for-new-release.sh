#!/bin/bash
 
existing_tags=$(skopeo list-tags docker://registry.access.redhat.com/container-native-virtualization/kubevirt-tekton-tasks-create-datavolume-rhel9 | jq '.Tags | sort | join(",")')

go run main.go --existing-tags="${existing_tags}" --minimal-version="v4.15" --dry-run=false --repository-url="https://github.com/ksimon1/tekton-tasks"