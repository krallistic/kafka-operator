#!/usr/bin/env bats

load "kubernetes_helper"


setup() {
  echo "Setup"
  kubectl apply -f kafka-operator.yaml
  sleep 60
}

teardown() {
  echo "Teardown"
  kubectl delete -f kafka-operator.yaml

}

@test "Test if operator is running" {
    wait_for_operator_running_or_fail
}



