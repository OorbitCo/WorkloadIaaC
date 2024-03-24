package main

import (
	"errors"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		region, regionOK := ctx.GetConfig("aws:region")
		windowsInstanceType, windowsInstanceTypeOK := ctx.GetConfig("worker:windowsInstance")
		linuxInstanceType, linuxInstanceTypeOK := ctx.GetConfig("worker:linuxInstance")
		windowsAMI, windowsAMIOK := ctx.GetConfig("worker:windowsAmi")
		if !regionOK || !windowsInstanceTypeOK || !linuxInstanceTypeOK || !windowsAMIOK {
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
		workloadCluster, err := eks.NewCluster(ctx, getStackName("WorkloadCluster", ctx.Stack()), &eks.ClusterArgs{
			AccessConfig: &eks.ClusterAccessConfigArgs{
				BootstrapClusterCreatorAdminPermissions: pulumi.Bool(true),
				AuthenticationMode:                      pulumi.String("API_AND_CONFIG_MAP"),
			},
			Name:    pulumi.String(getStackName("WorkloadCluster", ctx.Stack())),
			RoleArn: clusterRole.Arn,
			Tags: pulumi.StringMap{
				"Name": pulumi.String(getStackName("WorkloadCluster", ctx.Stack())),
			},
			Version: pulumi.String(K8S_VERSION),
			VpcConfig: &eks.ClusterVpcConfigArgs{
				EndpointPrivateAccess: pulumi.Bool(false),
				EndpointPublicAccess:  pulumi.Bool(true),
				VpcId:                 network.Vpc.ID(),
				SubnetIds:             network.getSubnetIds(),
				SecurityGroupIds: pulumi.StringArray{
					network.ClusterSecurityGroup.ID(),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{network.Vpc, clusterRole, network.ClusterSecurityGroup}))
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
		systemNodeGroup, err := eks.NewNodeGroup(ctx, getStackNameRegional("SystemNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &eks.NodeGroupArgs{
			ClusterName: workloadCluster.Name,
			NodeRoleArn: systemRole.Arn,
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			LaunchTemplate: &eks.NodeGroupLaunchTemplateArgs{
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
		linuxNodeGroup, err := eks.NewNodeGroup(ctx, getStackNameRegional("LinuxNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &eks.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("LinuxNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
			ClusterName:   workloadCluster.Name,
			NodeRoleArn:   linuxWorkerRole.Arn,
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			SubnetIds: network.getPublicSubnetIds(),
			InstanceTypes: pulumi.StringArray{
				pulumi.String(linuxInstanceType),
			},
			AmiType: pulumi.String("AL2_x86_64"),
			LaunchTemplate: &eks.NodeGroupLaunchTemplateArgs{
				Id:      linuxLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			Taints: eks.NodeGroupTaintArray{
				&eks.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_SCHEDULE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
				&eks.NodeGroupTaintArgs{
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
		windowsNodeGroup, err := eks.NewNodeGroup(ctx, getStackNameRegional("WindowsNodeGroup", ctx.Stack(), region, "WorkloadCluster"), &eks.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("WindowsNodeGroup", ctx.Stack(), region, "WorkloadCluster")),
			ClusterName:   workloadCluster.Name,
			NodeRoleArn:   winWorkerRole.Arn,
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
				MinSize:     pulumi.Int(1),
			},
			SubnetIds: network.getPublicSubnetIds(),
			LaunchTemplate: &eks.NodeGroupLaunchTemplateArgs{
				Id:      windowsLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			InstanceTypes: pulumi.StringArray{
				pulumi.String(windowsInstanceType),
			},
			Taints: eks.NodeGroupTaintArray{
				&eks.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_SCHEDULE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
				&eks.NodeGroupTaintArgs{
					Effect: pulumi.String("NO_EXECUTE"),
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("gpu"),
				},
			},
			Labels: pulumi.StringMap{
				"workload": pulumi.String("gpu"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, winWorkerRole, windowsLaunchTemplate}))
		ctx.Export("EKSCluster", workloadCluster.ID())
		ctx.Export("SystemNodeGroup", systemNodeGroup.ID())
		ctx.Export("LinuxNodeGroup", linuxNodeGroup.ID())
		ctx.Export("WindowsNodeGroup", windowsNodeGroup.ID())
		return nil
	})
}
