package main

import (
	"errors"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	awsEKS "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		region, regionOK := ctx.GetConfig("aws:region")
		windowsInstanceType, windowsInstanceTypeOK := ctx.GetConfig("worker:windowsInstance")
		linuxInstanceType, linuxInstanceTypeOK := ctx.GetConfig("worker:linuxInstance")
		windowsAMI, windowsAMIOK := ctx.GetConfig("worker:windowsAmi")
		adminUsername, adminOK := ctx.GetConfig("eks:adminUsername")
		accountId, accOK := ctx.GetConfig("eks:accountId")
		if !regionOK || !windowsInstanceTypeOK || !linuxInstanceTypeOK || !windowsAMIOK || !adminOK || !accOK {
			return errors.New("missing required configuration parameters")
		}
		ami, err := lookupAMI(ctx, windowsAMI)
		if err != nil {
			return err
		}
		network := new(Network)
		err = setupEKSNetwork(ctx, network)
		if err != nil {
			return err
		}
		clusterRole, err := createClusterRole(ctx, getStackNameRegional("ClusterRole", ctx.Stack(), region))
		if err != nil {
			return err
		}
		systemRole, err := createWorkerRole(ctx, getStackNameRegional("SystemRole", ctx.Stack(), region, "WorkloadCluster"))
		if err != nil {
			return err
		}
		winWorkerRole, err := createWorkerRole(ctx, getStackNameRegional("WindowsWorkerRole", ctx.Stack(), region, "WorkloadCluster"))
		if err != nil {
			return err
		}
		linuxWorkerRole, err := createWorkerRole(ctx, getStackNameRegional("LinuxWorkerRole", ctx.Stack(), region, "WorkloadCluster"))
		if err != nil {
			return err
		}
		workloadWorkerSecurityGroup, err := createWorkerSecurityGroup(ctx, network.Vpc)
		if err != nil {
			return err
		}
		workloadCluster, err := eks.NewCluster(ctx, getStackName("WorkloadCluster", ctx.Stack()), &eks.ClusterArgs{
			Name: pulumi.String(getStackName("WorkloadCluster", ctx.Stack())),
			InstanceRoles: iam.RoleArray{
				linuxWorkerRole,
				winWorkerRole,
			},
			VpcId:                        network.Vpc.ID(),
			PublicSubnetIds:              network.getPublicSubnetIds(),
			PrivateSubnetIds:             network.getPrivateSubnetIds(),
			Version:                      pulumi.String(K8S_VERSION),
			ServiceRole:                  clusterRole,
			NodeAssociatePublicIpAddress: new(bool),
			CreateOidcProvider:           pulumi.BoolPtr(true),
			ProviderCredentialOpts:       eks.KubeconfigOptionsArgs{},
			UserMappings: eks.UserMappingArray{
				&eks.UserMappingArgs{
					Groups:   pulumi.StringArray{pulumi.String("system:masters")},
					Username: pulumi.String(adminUsername),
					UserArn:  pulumi.String("arn:aws:iam::" + accountId + ":user/" + adminUsername),
				},
			},
			RoleMappings: eks.RoleMappingArray{
				&eks.RoleMappingArgs{
					Groups:   pulumi.StringArray{pulumi.String("system:bootstrappers"), pulumi.String("system:nodes"), pulumi.String("eks:kube-proxy-windows")},
					RoleArn:  winWorkerRole.Arn,
					Username: pulumi.String("system:node:{{EC2PrivateDNSName}}"),
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(getStackName("WorkloadCluster", ctx.Stack())),
			},
		}, pulumi.DependsOn([]pulumi.Resource{network.Vpc, clusterRole, network.ClusterSecurityGroup, clusterRole}))
		if err != nil {
			return err
		}
		systemLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("SystemLaunchTemplate", ctx.Stack(), region, "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			Name: pulumi.String(getStackNameRegional("SystemLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/xvda"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize:          pulumi.Int(100),
						VolumeType:          pulumi.String("gp3"),
						DeleteOnTermination: pulumi.String("true"),
					},
				},
			},
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("SystemLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
					},
				},
			},
		})
		systemNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("SystemNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			ClusterName: workloadCluster.EksCluster.Name(),
			NodeRoleArn: systemRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			LaunchTemplate: &awsEKS.NodeGroupLaunchTemplateArgs{
				Id:      systemLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			SubnetIds: network.getPrivateSubnetIds(),
			InstanceTypes: pulumi.StringArray{
				pulumi.String("t3.medium"),
			},
			Tags: pulumi.StringMap{
				"Name":     pulumi.String(getStackNameRegional("SystemNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
				"workload": pulumi.String("system"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, systemRole, systemLaunchTemplate}))
		linuxLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("LinuxLaunchTemplate", ctx.Stack(), region, "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			Name: pulumi.String(getStackNameRegional("LinuxLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
			VpcSecurityGroupIds: pulumi.StringArray{
				workloadWorkerSecurityGroup.ID(),
				network.ClusterSecurityGroup.ID(),
			},
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("LinuxLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
					},
				},
			},
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/xvda"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize: pulumi.Int(100),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadWorkerSecurityGroup, network.ClusterSecurityGroup}))
		linuxNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("LinuxNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("LinuxNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
			ClusterName:   workloadCluster.EksCluster.Name(),
			NodeRoleArn:   linuxWorkerRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			SubnetIds: network.getPublicSubnetIds(),
			InstanceTypes: pulumi.StringArray{
				pulumi.String(linuxInstanceType),
			},
			AmiType: pulumi.String("AL2_x86_64"),
			LaunchTemplate: &awsEKS.NodeGroupLaunchTemplateArgs{
				Id:      linuxLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			Taints: awsEKS.NodeGroupTaintArray{
				&awsEKS.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_SCHEDULE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
				&awsEKS.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_EXECUTE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
			},
			Labels: pulumi.StringMap{
				"workload": pulumi.String("gpu"),
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(getStackNameRegional("LinuxNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, linuxWorkerRole, linuxLaunchTemplate}))
		windowsLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("WindowsLaunchTemplate", ctx.Stack(), region, "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			Name:         pulumi.String(getStackNameRegional("WindowsLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
			ImageId:      pulumi.String(ami.ImageId),
			InstanceType: pulumi.String(windowsInstanceType),
			VpcSecurityGroupIds: pulumi.StringArray{
				workloadWorkerSecurityGroup.ID(),
				network.ClusterSecurityGroup.ID(),
			},
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("WindowsLaunchTemplate", ctx.Stack(), region, "WorkloadCluster")),
					},
				},
			},
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/xvda"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize:          pulumi.Int(100),
						VolumeType:          pulumi.String("gp3"),
						DeleteOnTermination: pulumi.String("true"),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadWorkerSecurityGroup, network.ClusterSecurityGroup}))
		windowsNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("WindowsNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("WindowsNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
			ClusterName:   workloadCluster.EksCluster.Name(),
			NodeRoleArn:   winWorkerRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			SubnetIds: network.getPublicSubnetIds(),
			LaunchTemplate: &awsEKS.NodeGroupLaunchTemplateArgs{
				Id:      windowsLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			InstanceTypes: pulumi.StringArray{
				pulumi.String(windowsInstanceType),
			},
			Taints: awsEKS.NodeGroupTaintArray{
				&awsEKS.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_SCHEDULE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
				&awsEKS.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_EXECUTE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
			},
			Labels: pulumi.StringMap{
				"workload": pulumi.String("gpu"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, winWorkerRole, windowsLaunchTemplate}))
		ctx.Export("EKSCluster", workloadCluster.Kubeconfig)
		ctx.Export("SystemNodeGroup", systemNodeGroup.ID())
		ctx.Export("LinuxNodeGroup", linuxNodeGroup.ID())
		ctx.Export("WindowsNodeGroup", windowsNodeGroup.ID())

		return nil
	})
}
