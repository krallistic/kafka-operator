# kafka-operator - A Kafka Operator for Kubernetes 

A Kubernetes Operator for Apache Kafka, which deploys, configures and manages your kafka cluster through its lifecycle. 
Features:
 - Fixed deployment of a Cluster, Services and PersistentVolumes
 - Upscaling of Cluster (eg adding a Broker)
 - Downscaling a Cluster, without dataloss (removes partition of broker first, under development)
 - 
 
Upcoming Features/Ideas: 
 - [] Vertical Pod Autoscaling
 - [] Managed Topics and hot partition detection/shuffling
 - [] Advanced Partition Shuffling (based on rate/size of incomming msg)


Currently the Operator is under development.
If you want to run Kafka in kubernetes a better option would be to look at the Helm Chart https://github.com/kubernetes/charts/blob/master/incubator/kafka/README.md alternative this: https://github.com/Yolean/kubernetes-kafka
 
# How to use it:

## 1.) Deploy the Operator
First we deploy the Operator inside our cluster:
```bash
# kubectl apply -f example/kafka-operator.yaml
deployment "kafka-operator" created
```

The Operator then creates a custom resource definition(CRD) "KafkaCluster" inside Kubernetes, which behaves like a normal k8s Object. 
The only difference is that no k8s internal components reacts to it, only our operator has a watch on it.

## 2) Deploy Zookeeper
Currently you need to deploy zookeeper by yourself (since managing zookeeper is a not a easy to topic, this is out of scope for now). As a starter you can find a example under `example/manual-zookeeper.yaml` for a single node zookeeper.
```bash
# kubectl apply -f example/manual-zookeeper.yaml
service "zk-headless" created
configmap "zk-config" created
statefulset "zk" created
```

## 3) Create a KafkaCluster spec and deploy
To deploy a kafka cluster we create spec (example/kafka-cluster.yaml): 

```yaml
apiVersion: "krallistic.github.com/v1"
kind: "Kafkacluster"
metadata:
  name: test-cluster-1
spec:
    brokerCount: 3
    topics:
      - name: "test1"
        replicationFactor: 1
        partitions: 1
      - name: "test2"
        replicationFactor: 2
        partitions: 2
    kafkaOptions:
       logRetentionHours: 24
       autoCreateTopics: false
       compressionType: "gzip"
    zookeeperConnect: zk-headless.default.svc.cluster.local
    image: confluentinc/cp-kafka:latest
    leaderImbalanceRatio: 0.1
    leaderImbalanceInterval: 600
    storageClass: emptyDir
    minimumGracePeriod: 1200
    jmxSidecar: false
    resources:
      cpu: "1"
      memory: "1Gi"
      diskSpace: "50G"

```
We can then just deploy this yaml via kubectl:
```bash
# kubectl apply -f example/kafka-cluster.yaml
kafkacluster "test-cluster-1" created
```
into kubernetes. This creates a ```kafkacluster``` object inside the api server. We can check this with:
```bash
# kubectl get kafkacluster
NAME             KIND
test-cluster-1   Kafkacluster.v1.krallistic.github.com
```
 
The operators then picks up the newly created object and creates the actual pods which are needed for the spezified Kafka cluster. 
Create the whole cluster can take a while but after a bit you should see every broker running and services created to either access direclty or all broker load-balanced:
```bash
# kubectl get pods,service
NAME                                                      READY     STATUS    RESTARTS   AGE
po/kafka-offset-checker-test-cluster-1-3029848613-z8rtd   1/1       Running   3          1m
po/kafka-operator-767603131-zcnt0                         1/1       Running   0          1m
po/test-cluster-1-0                                       1/1       Running   0          1m
po/test-cluster-1-1                                       1/1       Running   0          54s
po/test-cluster-1-2                                       1/1       Running   0          40s
po/zk-0                                                   1/1       Running   0          1m

NAME                          CLUSTER-IP     EXTERNAL-IP   PORT(S)             AGE
svc/kubernetes                10.7.240.1     <none>        443/TCP             5h
svc/test-cluster-1            None           <none>        9092/TCP            1m
svc/test-cluster-1-broker-0   10.7.243.30    <nodes>       9092:31545/TCP      1m
svc/test-cluster-1-broker-1   10.7.250.215   <nodes>       9092:31850/TCP      1m
svc/test-cluster-1-broker-2   10.7.249.221   <nodes>       9092:32653/TCP      1m
svc/zk-headless               None           <none>        2888/TCP,3888/TCP   1m
```

## 3) Resize the cluster
If we want to upscale the cluster we can just change the ```brokerCount``` value. 
After we changed it (for example to `5`) we do a ```kubectl apply -f example/kafka-cluster.yaml```. 
The operators then should pick up the change and start upsizing the cluster: 
```bash
# kubectl apply -f example/kafka-cluster.yaml
kafkacluster "test-cluster-1" configured
kubectl get pods
NAME                                                   READY     STATUS    RESTARTS   AGE
kafka-offset-checker-test-cluster-1-3029848613-z8rtd   1/1       Running   3          4m
kafka-operator-767603131-zcnt0                         1/1       Running   0          4m
test-cluster-1-0                                       1/1       Running   0          4m
test-cluster-1-1                                       1/1       Running   0          4m
test-cluster-1-2                                       1/1       Running   0          3m
test-cluster-1-3                                       0/1       Pending   0          35s
zk-0                                                   1/1       Running   0          4m
```
NOTE: Currently the operator does not automaticly rebalance topics with the new broker

### 3.b) Downscaling:
While downscaling the cluster is possible and simple a simple rebalancing is done to prevent data-loss, this is currently heavy under development and considered unstable.

## 4) Delete the cluster
When we are done, we can do a
```bash
# kubectl delete -f example/kafka-cluster.yaml
kafkacluster "test-cluster-1" deleted
```
to delete the `kafkaCluster` object.
The operator then detects the deletion and shuts down all running components:
```bash
# kubectl get pods
NAME                             READY     STATUS        RESTARTS   AGE
kafka-operator-767603131-tv3ck   1/1       Running       0          1m
test-cluster-1-0                 0/1       Terminating   0          8m
zk-0                             1/1       Running       0          9m
```

# Known Issues / Open Tasks
There are a couple of open Task/Issues, this is mainly just for me tracking progress:

- [ ] Resisze Clusters (without Data Rebalancing)
- [x] Delete Cluster
- [ ] Dokumentation, Vendoring and Testing
- [ ] Use Ressource (K8s and kafka Options)
- [ ] Monitoring with JMX Sidecar
- [ ] Automaticly Rebalacing
- [ ] Investigate Datagravity 


### Zookeeper
To get Kafka Running a Zookeeper is needed. A simple one Node zK example is provided in the example Folder. But for any usage beyond testing/developing a proper Zookeeper setup should be used. A good example is the Zookeeper Chart in the offical Helm Repo.


# Details

## Differences vs Helm Chart
While a Helm is a great tool, and the provided kafka chart is also pretty dope, Helm only managees deployment of a cluster. But since kafka is a statefull application its needs goes beyond the normal capabilities you can do with vanilla kubernetes. For example a downsizing/upsizing of the cluster requires moving partitions/replicas off/onto brokers. To automate that the operator is used. It looks at the current cluster state and takes neccesary actions to 

## Images:
Currently the supported image is are the offical images from confluent. (https://github.com/confluentinc/cp-docker-images) While its possible to specify other images, due to instrumentation most other images wont work. 

# Development

## Dependency Managment

```dep``` is used for dependecy managment. 

Under `e2e-test/hack` are a couple of