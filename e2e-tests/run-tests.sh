#!/usr/bin/env bash

source hack/setup-gcloud-cluster.sh

bats 01-test-basic-setup.bats

source hack/delete-gcloud-cluser.sh