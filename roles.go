package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createClusterRole(ctx *pulumi.Context, roleName string) (*iam.Role, error) {
	clusterRole, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {
							"Service": "eks.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
		// Attach the AmazonEksCluster and AmazonEksVpcResourceController policies
		ManagedPolicyArns: pulumi.StringArray{
			pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
			pulumi.String("arn:aws:iam::aws:policy/AmazonEKSVPCResourceController"),
		},
	})
	return clusterRole, err
}
func createWorkerRole(ctx *pulumi.Context, roleName string) (*iam.Role, error) {
	workerRole, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {
							"Service": "ec2.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
		// Attach the AmazonEKSWorkerNodePolicy, AmazonEKS_CNI_Policy, and AmazonEC2ContainerRegistryReadOnly policies
		ManagedPolicyArns: pulumi.StringArray{
			pulumi.String("arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"),
			pulumi.String("arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"),
			pulumi.String("arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"),
			pulumi.String("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		},
	})
	return workerRole, err
}
