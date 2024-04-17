package main

const (
	BASE_STACK_NAME = "workload"
	K8S_VERSION     = "1.29"
)

var stackName string
var region string

func getStackNameRegional(args ...string) string {
	name := BASE_STACK_NAME
	for _, arg := range args {
		name += "-" + arg
	}
	name += "-" + region
	name += "-" + stackName
	return name
}
