module k8s.io/kms/plugins/mock

go 1.19

require (
	k8s.io/klog/v2 v2.90.1
	k8s.io/kms v0.0.0-00010101000000-000000000000
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	k8s.io/client-go v0.27.1 // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
)

replace (
	k8s.io/apimachinery => ../../../../apimachinery
	k8s.io/client-go => ../../../../client-go
	k8s.io/kms => ../../../../kms
)

replace github.com/onsi/ginkgo/v2 => github.com/openshift/onsi-ginkgo/v2 v2.6.1-0.20230317131656-c62d9de5a460
