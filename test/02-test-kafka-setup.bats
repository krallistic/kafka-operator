#!/usr/bin/env bats

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
  kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase'
  $name=kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .metadata.name'
  run bash -c "kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase'"
  echo $name
  echo $output
  [ "$output" = "Running" ]
}



