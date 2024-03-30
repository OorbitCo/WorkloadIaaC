package main

import (
	"fmt"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const template = `<powershell>
[string]$EKSBootstrapScriptFile = "$env:ProgramFiles\Amazon\EKS\Start-EKSBootstrap.ps1"
net user WIN_GPU_INSTANCE_USERNAME_PLACEHOLDER "WIN_GPU_INSTANCE_PASSWORD_PLACEHOLDER"
& $EKSBootstrapScriptFile -EKSClusterName "%s" -APIServerEndpoint "%s" -Base64ClusterCA "%s" -DNSClusterIP "172.20.0.10" -ContainerRuntime "containerd" -KubeletExtraArgs "--node-labels=" 3>&1 4>&1 5>&1 6>&1
</powershell>`

func getWindowsUserData(cluster *eks.Cluster) pulumi.StringPtrInput {
	clusterName := cluster.EksCluster.Name()
	endpoint := cluster.EksCluster.Endpoint()
	certificateAuthorityData := cluster.EksCluster.CertificateAuthority().Data()
	combined := pulumi.All(clusterName, endpoint, certificateAuthorityData).ApplyT(func(args []interface{}) (string, error) {
		userData := fmt.Sprintf(template, args[0], args[1], args[2])
		fmt.Printf("Windows user data: %s\n", userData)
		return userData, nil
	})
	return combined.ApplyT(func(userData string) *string { return &userData }).(pulumi.StringPtrInput)
}
