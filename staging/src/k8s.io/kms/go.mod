// This is a generated file. Do not edit directly.

module k8s.io/kms

go 1.19

require (
	github.com/gogo/protobuf v1.3.2
	google.golang.org/grpc v1.56.3
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace (
	golang.org/x/net => github.com/openshift-priv/golang-net v0.0.0-20240328080036-60d0f00ca866
	k8s.io/kms => ../kms
)
