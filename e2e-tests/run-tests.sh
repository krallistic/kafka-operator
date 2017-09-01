#!/usr/bin/env bash
echo "Setting Up Cluster"
source hack/setup-gcloud-cluster.sh

echo "Running BATS Tests"
bats 01-test-basic-setup.bats
bats 02-test-kafka-setup.bats

echo "Destroying Cluster"
source hack/delete-gcloud-cluser.sh