# kafka-operator - [WIP] A Kafka Operator for Kubernetes 

Currently the Operator is under development. Currently only the bare minimum of running a StatefulSet. Not even all arguments in the spec are supported.
 
There is no tested image for running in inside a cluster. All test are done by running it standalone and use `kubectl proxy -p 8080` to map the API to `localhost:8080'.
The operator creates a ThirdPartyRessources "KafkaCluster" inside kubernete. You can then create kafka Cluster by using the KafkaCluster Objekt. 

# Example

## 1) Deploy the Operator
First we deploy the Operator (Status: March 2017, only tested as  outOfCluster Configuration).

The Operator then creates a third party ressource "KafkaCluster" inside Kubernetes, which behaves like a normal k8s Object. 
The only difference is that no k8s internal components reacts to it, only our operator has a watch on it.

## 2) Create a KafkaCluster spec and deploy
To deploy a kafka cluster we create spec (example/kafkaObj.yaml): 

```yaml
apiVersion: "incubator.test.com/v1"
kind: "KafkaCluster"
metadata:
  name: hello-world-cluster-2
spec:
    name: operator
    brokerCount: 2
    brokers:
        - brokerID: 1
          memory: 512
          diskSpace: 25
          cpu: 0.5
          topics:
        - brokerID: 2
          memory: 1024
          diskSpace: 50
          cpu: 1
    kafkaOptions:
       logRetentionHours: 24
    zookeeperConnect: zk-headless.default.svc.cluster.local
    image: confluentinc/cp-kafka:latest
    storageClass: emptyDir
    jmxSidecar: false
```
We can then just deploy this yaml via kubectl:
```kubectl create -f examples/kafkaObj.yaml```
into kubernetes. This creates a ```kafkaCluster``` object inside the api server. We can check this with:
```
kubectl get kafkaCluster
NAME                    KIND
hello-world-cluster-2   KafkaCluster.v1.incubator.test.com
```
 
The operators then picks up the newly created object and creates the actual pods which are needed for the spezified Kafka cluster. 
After some seconds we can see the created pods running: 
```
kubectl get pods
NAME         READY     STATUS    RESTARTS   AGE
operator-0   1/1       Running   0          21s
operator-1   1/1       Running   0          18s
```

## 3) Resize the cluster
If we want to upscale the cluster we can just change the ```brokerCount``` value. 
After we changed it (for example to ```3```) we do a ```kubectl apply -f example/kafkaObj.yaml```. 
The operators then should pick up the change an create another broker. We can check this with:
```
TODO kubectl ouput
```

NOTE: Currently the operator does not automaticly rebalance topics with the new broker

NOTE: Downscaling is currently not supported since its unclear how to handle the data.

## 4) Delete the cluster
When we are done, we can do a
```
kubectl delete -f example/kafkaObj.yaml
kafkacluster "hello-world-cluster-2" deleted
```
to delete the `kafkaCluster` object.
The operator then detects the deletion and shuts down all running components. 
```
kubectl get pods
NAME      READY     STATUS    RESTARTS   AGE
zk-0      1/1       Running   2          7d
```

# Known Issues / Open Tasks
There are a couple of open Task/Issues, this is mainly just for me tracking progress:

- [ ] Resisze Clusters (without Data Rebalancing)
- [ ] Delete Cluster
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
