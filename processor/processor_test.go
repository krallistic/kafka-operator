package processor

import (
	"reflect"
	"testing"

	"github.com/krallistic/kafka-operator/controller"
	"github.com/krallistic/kafka-operator/kafka"
	spec "github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"
	k8sclient "k8s.io/client-go/kubernetes"
)

func TestProcessor_DetectChangeType(t *testing.T) {
	type fields struct {
		client          k8sclient.Clientset
		baseBrokerImage string
		util            util.ClientUtil
		tprController   controller.CustomResourceController
		kafkaClusters   map[string]*spec.Kafkacluster
		watchEvents     chan spec.KafkaclusterWatchEvent
		clusterEvents   chan spec.KafkaclusterEvent
		kafkaClient     map[string]*kafka.KafkaUtil
		control         chan int
		errors          chan error
	}
	type args struct {
		event spec.KafkaclusterWatchEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   spec.KafkaclusterEvent
	}{
		{
			"detect error state change if everything is nil",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{},
			},
			spec.KafkaclusterEvent{Type: spec.ERROR_STATE, Cluster: spec.Kafkacluster{}},
		},
		{
			"detect if state is changed",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{
					Type: "TEST",
					Object: spec.Kafkacluster{
						State: spec.KafkaclusterState{
							Status: "stateAfter",
						},
					},
					OldObject: spec.Kafkacluster{
						State: spec.KafkaclusterState{
							Status: "stateBefore",
						},
					},
				},
			},
			spec.KafkaclusterEvent{
				Type: spec.STATE_CHANGE,
				Cluster: spec.Kafkacluster{
					State: spec.KafkaclusterState{
						Status: "stateAfter",
					},
				},
				OldCluster: spec.Kafkacluster{
					State: spec.KafkaclusterState{
						Status: "stateBefore",
					},
				},
			},
		},
		{
			"replica count changed, upscale",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{
					Type: "TEST",
					Object: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 3,
						},
					},
					OldObject: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 2,
						},
					},
				},
			},
			spec.KafkaclusterEvent{
				Type: spec.UPSIZE_CLUSTER,
				Cluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 3,
					},
				},
				OldCluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 2,
					},
				},
			},
		},
		{
			"replica count changed, downsclale",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{
					Type: "TEST",
					Object: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 2,
						},
					},
					OldObject: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 3,
						},
					},
				},
			},
			spec.KafkaclusterEvent{
				Type: spec.DOWNSIZE_CLUSTER,
				Cluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 2,
					},
				},
				OldCluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 3,
					},
				},
			},
		},
		{
			"add cluster event",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{
					Type: "ADDED",
					Object: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 2,
						},
					},
				},
			},
			spec.KafkaclusterEvent{
				Type: spec.NEW_CLUSTER,
				Cluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 2,
					},
				},
			},
		},
		{
			"delete cluster event",
			fields{},
			args{
				event: spec.KafkaclusterWatchEvent{
					Type: "DELETED",
					Object: spec.Kafkacluster{
						Spec: spec.KafkaclusterSpec{
							BrokerCount: 2,
						},
					},
				},
			},
			spec.KafkaclusterEvent{
				Type: spec.DELETE_CLUSTER,
				Cluster: spec.Kafkacluster{
					Spec: spec.KafkaclusterSpec{
						BrokerCount: 2,
					},
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Processor{
				client:          tt.fields.client,
				baseBrokerImage: tt.fields.baseBrokerImage,
				util:            tt.fields.util,
				tprController:   tt.fields.tprController,
				kafkaClusters:   tt.fields.kafkaClusters,
				watchEvents:     tt.fields.watchEvents,
				clusterEvents:   tt.fields.clusterEvents,
				kafkaClient:     tt.fields.kafkaClient,
				control:         tt.fields.control,
				errors:          tt.fields.errors,
			}
			if got := p.DetectChangeType(tt.args.event); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Processor.DetectChangeType() = %v, want %v", got, tt.want)
			}
		})
	}
}
