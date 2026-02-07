module github.com/paasdeploy/agent

go 1.24.0

toolchain go1.24.12

require (
	github.com/paasdeploy/backend v0.0.0
	github.com/paasdeploy/shared v0.0.0
	golang.org/x/sys v0.40.0
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
)

replace (
	github.com/paasdeploy/backend => ../backend
	github.com/paasdeploy/shared => ../shared
)
