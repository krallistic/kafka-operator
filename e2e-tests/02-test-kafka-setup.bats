#!/usr/bin/env bats

load "hack/kubernetes_helper"


#Global Suite setup, bats dont support these, so hack with BATS_TESTNumer
suite-setup() {
  if [ "$BATS_TEST_NUMBER" -eq 1 ]; then
    echo "Global Setup"
    kubectl apply -f files/manual-zookeeper.yaml
    kubectl apply -f files/kafka-operator.yaml
    wait_for_operator_running_or_fail
    wait_for_zookeeper_running_or_fail
    kubectl apply -f files/02-basic-cluster.yaml
    wait_for_brokers_running_or_fail 3
  fi
}

suite-teardown() {
  if [ "$BATS_TEST_NUMBER" -eq 3 ]; then
    echo "Global Teardown"
    kubectl delete -f files/02-basic-cluster.yaml
    kubectl delete -f files/manual-zookeeper.yaml
    #TODO wait till sts is fully deleted
    kubectl delete -f files/kafka-operator.yaml
    kubectl delete statefulset test-cluster-1
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
  suite-teardown
}

@test "Test if headless service is created" {
    run kubectl get svc test-cluster-1
    [ "$status" -eq 0 ]
}

@test "Test if direct Broker are created" {
    run kubectl get svc test-cluster-1-broker-0
    [ "$status" -eq 0 ]
    run kubectl get svc test-cluster-1-broker-1
    [ "$status" -eq 0 ]
    run kubectl get svc test-cluster-1-broker-2
    [ "$status" -eq 0 ]
}

@test "test brokers are created and running" {
  wait_for_brokers_running_or_fail 3
}



