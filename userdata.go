package main

import (
	"encoding/base64"
	"fmt"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"strings"
)

// net user WIN_GPU_INSTANCE_USERNAME_PLACEHOLDER "WIN_GPU_INSTANCE_PASSWORD_PLACEHOLDER"
const template = `<powershell>
& "C:\Program Files\Amazon\EKS\Start-EKSBootstrap.ps1" -EKSClusterName "%s" -APIServerEndpoint "%s" -Base64ClusterCA "%s" -DNSClusterIP "%s" -ContainerRuntime "containerd" -KubeletExtraArgs "--node-labels=" 3>&1 4>&1 5>&1 6>&1
</powershell>`

func getWindowsUserData(ctx *pulumi.Context, cluster *eks.Cluster, clusterIP pulumi.Output) pulumi.StringPtrInput {
	clusterName := cluster.EksCluster.Name()
	endpoint := cluster.EksCluster.Endpoint()
	certificateAuthorityData := cluster.EksCluster.CertificateAuthority().Data()
	combined := pulumi.All(clusterName, endpoint, certificateAuthorityData, clusterIP).ApplyT(func(args []interface{}) (string, error) {
		certificate := *args[2].(*string)
		certificate = strings.ReplaceAll(certificate, "\n", "")
		certificate = strings.ReplaceAll(certificate, "\r", "")
		userData := fmt.Sprintf(template, args[0], args[1], certificate, *args[3].(*string))
		ctx.Log.Debug(fmt.Sprintf("Windows user data: %s\n", userData), nil)
		userData = base64.StdEncoding.EncodeToString([]byte(userData))
		return userData, nil
	})
	return combined.ApplyT(func(userData string) *string { return &userData }).(pulumi.StringPtrInput)

}
