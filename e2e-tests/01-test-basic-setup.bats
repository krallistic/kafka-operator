#!/usr/bin/env bats

load "hack/kubernetes_helper"


setup() {
  echo "Setup"
  kubectl apply -f files/kafka-operator.yaml
    wait_for_operator_running_or_fail
}

teardown() {
  echo "Teardown"
  kubectl delete -f files/kafka-operator.yaml

}

@test "Test if operator is running" {
    wait_for_operator_running_or_fail
}



