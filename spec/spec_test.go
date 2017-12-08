package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ runtime.Object = &Kafkacluster{}
var _ metav1.ObjectMetaAccessor = &Kafkacluster{}

var _ runtime.Object = &KafkaclusterList{}
var _ metav1.ListMetaAccessor = &KafkaclusterList{}

// func exampleFuzzerFuncs(t apitesting.TestingCommon) []interface{} {
// 	return []interface{}{
// 		func(obj *KafkaclusterList, c fuzz.Continue) {
// 			c.FuzzNoCustom(obj)
// 			obj.Items = make([]Kafkacluster, c.Intn(10))
// 			for i := range obj.Items {
// 				c.Fuzz(&obj.Items[i])
// 			}
// 		},
// 	}
// }

// // TestRoundTrip tests that the third-party kinds can be marshaled and unmarshaled correctly to/from JSON
// // without the loss of information. Moreover, deep copy is tested.
// func TestRoundTrip(t *testing.T) {
// 	scheme := runtime.NewScheme()
// 	codecs := serializer.NewCodecFactory(scheme)

// 	AddToScheme(scheme)

// 	seed := rand.Int63()
// 	fuzzerFuncs := fuzzer.MergeFuzzerFuncs(t, fuzzer.GenericFuzzerFuncs(t, codecs), exampleFuzzerFuncs(t))
// 	fuzzer := fuzzer.FuzzerFor(fuzzerFuncs, rand.NewSource(seed))

// 	roundtrip.RoundTripSpecificKindWithoutProtobuf(t, SchemeGroupVersion.WithKind("Kafkacluster"), scheme, codecs, fuzzer, nil)
// 	roundtrip.RoundTripSpecificKindWithoutProtobuf(t, SchemeGroupVersion.WithKind("KafkaclusterList"), scheme, codecs, fuzzer, nil)
// }
