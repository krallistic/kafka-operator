# kafka-operator - [WIP] A Kafka Operator for Kubernetes 

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
This object is then picked up by the Operator and lets him create a StatefulSet for the Brokers. 

# Known Issues / Open Tasks
There are a couple of open Task/Issues, this is mainly just for me tracking progress:

- [ ] Resisze Clusters (without Data Rebalancing)
- [x] Delete Cluster
- [ ] Dokumentation, Vendoring and Testing
- [ ] Use Ressource (K8s and kafka Options)
- [ ] Monitoring with JMX Sidecar
- [ ] Automaticly Rebalacing
- [ ] Investigate Datagravity 


Under kubectl 1.5.X there is an issue with apply and edit on tpr ressources, this is fixed on 1.6, so user should use a these (you can still manually POST stuff to the API and it will work).


### Zookeeper
To get Kafka Running a Zookeeper is needed. A simple one Node zK example is provided in the example Folder. But for any usage beyond testing/developing a proper Zookeeper setup should be used. A good example is the Zookeeper Chart in the offical Helm Repo.


# Details



## Images:
Currently the supported image is are the offical images from confluent. (https://github.com/confluentinc/cp-docker-images) While its possible to specify other images, due to instrumentation most other images wont work. 

## Naming Scheme:

The defualt TPR has a Name which serves a the domain Name of the Cluster, the sts object name is by default just plain Kafka. TODO make Configurable
With a KafkaCluster with the Name "advertising" this resolves to the following name:
`kafka-0.advertising.default.cluster.local`
