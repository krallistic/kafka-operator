#!/usr/bin/env bash

gcloud config set compute/zone europe-west1-d


gcloud container clusters create kafka-operator-test-cluster \
    --num-nodes 3 \
    --machine-type n1-standard-2 \
    --scopes storage-rw \
    --preemptible \
    --cluster-version=1.7.6-gke.1 \
    --no-async \
    --enable-kubernetes-alpha

gcloud container clusters get-credentials kafka-operator-test-cluster
