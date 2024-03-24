package main

const (
	BASE_STACK_NAME = "workload"
	K8S_VERSION     = "1.29"
)

func getStackName(args ...string) string {
	return getStackNameRegional(args...)
}
func getStackNameRegional(args ...string) string {
	name := BASE_STACK_NAME
	for _, arg := range args {
		name += "-" + arg
	}
	return name
}
