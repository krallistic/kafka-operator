# kafka-operator
[WIP] A Kafka Operator for Kubernetes 


Its pretty basic

# Naming Sheme:

The defualt TPR has a Name which serves a the domain Name of the Cluster, the sts object name is by default just plain Kafka. TODO make Configurable
With a KafkaCluster with the Name "advertising" this resolves to the following name:
kafka-0.advertising.default.cluster.local






# Develop
Basic Architecture inspired from the great work of Steve Sloka <slokas@upmc.edu> on the elasticsearch operator