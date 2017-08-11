#!/usr/bin/env bats

load "kubernetes_helper"


#Global Suite setup, bats dont support these, so hack with BATS_TESTNumer
suite-setup() {
  if [ "$BATS_TEST_NUMBER" -eq 1 ]; then
    echo "Global Setup"
    kubectl apply -f manualZookeeper.yaml
    kubectl apply -f kafka-operator.yaml
    wait_for_operator_running_or_fail
    wait_for_zookeeper_running_or_fail
    kubectl apply -f 02-basic-cluster.yaml
    wait_for_brokers_running_or_fail 3
  fi
}

suite-teardown() {
  if [ "$BATS_TEST_NUMBER" -eq 4 ]; then
    echo "Global Setup"
    kubectl delete -f manualZookeeper.yaml
    kubectl delete -f kafka-operator.yaml
    wait_for_operator_running_or_fail
    wait_for_zookeeper_running_or_fail
    kubectl delete -f 02-basic-cluster.yaml
    wait_for_brokers_running_or_fail
  fi
}

setup() {
 #Empty 
 suite-setup
 echo "Setup"
}

teardown() {
  #Empty for now since we want a global setup
  echo "Teardown"
}

@test "Headless service is created" {
    run kubectl get svc test-cluster-2
    [ "$status" -eq 1 ]
}

@test "Direct Broker are created" {
    run kubectl get svc broker-0
    [ "$status" -eq 1 ]
    run kubectl get svc broker-1
    [ "$status" -eq 1 ]
    run kubectl get svc broker-2
    [ "$status" -eq 1 ]
}

@test "Brokers are created" {
  wait_for_brokers_running_or_fail 3
}

@test "Test if operator is running" {
  kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase'
  $name=kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .metadata.name'
  run bash -c "kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase'"
  echo $name
  echo $output
  [ "$output" = "Running" ]
}



