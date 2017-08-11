#!/usr/bin/env bash

is_operator_running() {
    PHASE=$(kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase')
    EXPECTED='Running'
    if [ "$EXPECTED" = "$PHASE" ]
    then
      return true
    else
      return false
    fi
}

wait_for_operator_running_or_fail() {
    for try in {1..10} ; do
        PHASE=$(kubectl get pod -l name=kafka-operator,type=operator -ojson | jq -r '.items[] | .status.phase')
        EXPECTED='Running'
        if [ "$EXPECTED" = "$PHASE" ]
        then
          return 0
        else
          sleep 10
        fi
    done
    echo "Waited for 100 seconds, operator not ready"
    return 1
}

wait_for_zookeeper_running_or_fail() {
    for try in {1..10} ; do
        PHASE=$(kubectl get pod -l app=zk -ojson | jq -r '.items[] | .status.phase')
        EXPECTED='Running'
        if [ "$EXPECTED" = "$PHASE" ]
        then
          return 0
        else
          sleep 10
        fi
    done
    echo "Waited for 100 seconds, zookeeper not ready"
    return 1
}

wait_for_broker_X_running_or_fail() {
    echo "Waiting till broker $1 is ready"
    for try in {1..10} ; do
        PHASE=$(kubectl get pod -l creator=kafkaOperator,kafka_broker_id=$1 -ojson | jq -r '.items[] | .status.phase')
        EXPECTED='Running'
        if [ "$EXPECTED" = "$PHASE" ]
        then
          return 0
        else
          sleep 10
        fi
    done
    echo "Waited for 100 seconds, Broker not ready"
    return 1
}

wait_for_brokers_running_or_fail() {
    for id in {0..$1} ; do
        wait_for_broker_X_running_or_fail $id
        if [ "$?" -eq 1 ]
        then
            echo "Waited to long for Broker $id"
            return 1
        fi
    done
    return 0
}
