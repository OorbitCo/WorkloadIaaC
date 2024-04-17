package main

import (
	"encoding/base64"
	"fmt"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"strings"
)

// net user WIN_GPU_INSTANCE_USERNAME_PLACEHOLDER "WIN_GPU_INSTANCE_PASSWORD_PLACEHOLDER"
// & "C:\Program Files\Amazon\EKS\Start-EKSBootstrap.ps1" -EKSClusterName "%s" -APIServerEndpoint "%s" -Base64ClusterCA "%s" -DNSClusterIP "%s" -ContainerRuntime "containerd" -KubeletExtraArgs "--node-labels=" 3>&1 4>&1 5>&1 6>&1
// $drive_letter = "C"
// $size = (Get-PartitionSupportedSize -DriveLetter $drive_letter)
// Resize-Partition -DriveLetter $drive_letter -Size $size.SizeMax
const windowsTemplate = `<powershell>
net user Administrator Sup3rs3cret!!!
[string]$EKSBootstrapScriptFile = "$env:ProgramFiles\Amazon\EKS\Start-EKSBootstrap.ps1"
& $EKSBootstrapScriptFile -EKSClusterName %s -APIServerEndpoint %s -Base64ClusterCA %s -DNSClusterIP %s -ContainerRuntime containerd -KubeletExtraArgs "--node-labels=" 3>&1 4>&1 5>&1 6>&1
</powershell>
<persist>true</persist>
`

const linuxTemplate = `#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh %s --apiserver-endpoint %s --b64-cluster-ca %s --dns-cluster-ip %s --container-runtime containerd --kubelet-extra-args "--node-labels="`

func getWindowsUserData(ctx *pulumi.Context, cluster *eks.Cluster, clusterIP pulumi.Output) pulumi.StringPtrInput {
	clusterName := cluster.EksCluster.Name()
	endpoint := cluster.EksCluster.Endpoint()
	certificateAuthorityData := cluster.EksCluster.CertificateAuthority().Data()
	combined := pulumi.All(clusterName, endpoint, certificateAuthorityData, clusterIP).ApplyT(func(args []interface{}) (string, error) {
		certificate := *args[2].(*string)
		certificate = strings.ReplaceAll(certificate, "\n", "")
		certificate = strings.ReplaceAll(certificate, "\r", "")
		userData := fmt.Sprintf(windowsTemplate, args[0], args[1], certificate, *args[3].(*string))
		ctx.Log.Debug(fmt.Sprintf("Windows user data: %s\n", userData), nil)
		userData = base64.StdEncoding.EncodeToString([]byte(userData))
		return userData, nil
	})
	return combined.ApplyT(func(userData string) *string { return &userData }).(pulumi.StringPtrInput)
}

func getLinuxUserData(ctx *pulumi.Context, cluster *eks.Cluster, clusterIP pulumi.Output) pulumi.StringPtrInput {
	clusterName := cluster.EksCluster.Name()
	endpoint := cluster.EksCluster.Endpoint()
	certificateAuthorityData := cluster.EksCluster.CertificateAuthority().Data()
	combined := pulumi.All(clusterName, endpoint, certificateAuthorityData, clusterIP).ApplyT(func(args []interface{}) (string, error) {
		certificate := *args[2].(*string)
		certificate = strings.ReplaceAll(certificate, "\n", "")
		certificate = strings.ReplaceAll(certificate, "\r", "")
		userData := fmt.Sprintf(linuxTemplate, args[0], args[1], certificate, *args[3].(*string))
		ctx.Log.Debug(fmt.Sprintf("Linux user data: %s\n", userData), nil)
		userData = base64.StdEncoding.EncodeToString([]byte(userData))
		return userData, nil
	})
	return combined.ApplyT(func(userData string) *string { return &userData }).(pulumi.StringPtrInput)
}
