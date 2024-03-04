#!/bin/bash

rm -r kubevirt-tekton-tasks || true

git clone git@github.com:kubevirt/kubevirt-tekton-tasks.git
cd kubevirt-tekton-tasks || exit 1


cp -r "../ansible" "scripts/ansible/"

find . -type f -name "*.yaml" -print0 | xargs -0 sed -i "s/quay.io\/kubevirt\/tekton-tasks/some_different_image/g"
find . -type f -name "*.yaml" -print0 | xargs -0 sed -i "s/quay.io\/kubevirt\/tekton-tasks-disk-virt/some_image/g"

make generate-yaml-tasks
make generate-pipelines
#delete tasks, which are not published
for TASK_NAME in "execute-in-vm" "generate-ssh-keys"
do
	rm -r "tasks/${TASK_NAME}"
done

../run-catalog-cd.sh
