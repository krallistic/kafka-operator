# kafka-operator
[WIP] A Kafka Operator for Kubernetes 

Currently the Operator is under development. Currently only the bare minimum of running a StatefulSet. Not even all arguments in the spec are supported.
 
There is no tested image for running in inside a cluster. All test are done by running it standalone and use `kubectl proxy -p 8080` to map the API to `localhost:8080'.
The operator creates a ThirdPartyRessources "KafkaCluster" inside kubernete. You can then create kafka Cluster by using the KafkaCluster Objekt. 

```yaml
apiVersion: "incubator.test.com/v1"
kind: "KafkaCluster"
metadata:
  name: hello-world-cluster-2
spec:
    name: operator
    brokers:
        count: 2
        memory: 1024
        diskSpace: 50
        cpu: 1
    kafkaOptions:
       logRetentionHours: 24
    zookeeperConnect: zk-headless.default.svc.cluster.local
    image: confluentinc/cp-kafka:latest
    jmxSidecar: false
```
This object is then picked up by the Operator and lets him create a StatefulSet for the Brokers


# Details



## Images:
Currently the supported image is are the offical images from confluent. While its possible to specify other images, due to instrumentation most other images wont work. 

## Naming Scheme:

The defualt TPR has a Name which serves a the domain Name of the Cluster, the sts object name is by default just plain Kafka. TODO make Configurable
With a KafkaCluster with the Name "advertising" this resolves to the following name:
`kafka-0.advertising.default.cluster.local`
