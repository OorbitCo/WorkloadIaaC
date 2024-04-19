package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	awsEKS "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v2/go/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
	"strconv"
)

func StringPtr(s string) *string {
	return &s
}
func validate(regionOK bool, windowsInstanceTypeOK bool, linuxInstanceTypeOK bool, windowsAMIOK bool, adminOK bool, accOK bool, linuxDesiredCapacityOK bool, linuxMinSizeOK bool, linuxMaxSizeOK bool, windowsDesiredCapacityOK bool, windowsMinSizeOK bool, windowsMaxSizeOK bool, windowsPasswordOK bool) error {
	if !regionOK {
		return errors.New("region is required")
	}
	if !windowsInstanceTypeOK {
		return errors.New("windowsInstance is required")
	}
	if !linuxInstanceTypeOK {
		return errors.New("linuxInstance is required")
	}
	if !windowsAMIOK {
		return errors.New("windowsAmi is required")
	}
	if !adminOK {
		return errors.New("adminUsername is required")
	}
	if !accOK {
		return errors.New("accountId is required")
	}
	if !linuxDesiredCapacityOK {
		return errors.New("linuxDesiredCapacity is required")
	}
	if !linuxMinSizeOK {
		return errors.New("linuxMinSize is required")
	}
	if !linuxMaxSizeOK {
		return errors.New("linuxMaxSize is required")
	}
	if !windowsDesiredCapacityOK {
		return errors.New("windowsDesiredCapacity is required")
	}
	if !windowsMinSizeOK {
		return errors.New("windowsMinSize is required")
	}
	if !windowsMaxSizeOK {
		return errors.New("windowsMaxSize is required")
	}
	if !windowsPasswordOK {
		return errors.New("windowsPassword is required")
	}
	return nil
}
func main() {
	falsePtr := new(bool)
	truePtr := new(bool)
	*falsePtr = false
	*truePtr = true
	pulumi.Run(func(ctx *pulumi.Context) error {
		rg, regionOK := ctx.GetConfig("aws:region")
		windowsInstanceType, windowsInstanceTypeOK := ctx.GetConfig("worker:windowsInstance")
		linuxInstanceType, linuxInstanceTypeOK := ctx.GetConfig("worker:linuxInstance")
		windowsAMI, windowsAMIOK := ctx.GetConfig("worker:windowsAmi")
		adminUsername, adminOK := ctx.GetConfig("eks:adminUsername")
		accountId, accOK := ctx.GetConfig("eks:accountId")
		linuxDesiredCapacity, linuxDesiredCapacityOK := ctx.GetConfig("worker:linuxDesiredCapacity")
		linuxMinSize, linuxMinSizeOK := ctx.GetConfig("worker:linuxMinSize")
		linuxMaxSize, linuxMaxSizeOK := ctx.GetConfig("worker:linuxMaxSize")

		windowsDesiredCapacity, windowsDesiredCapacityOK := ctx.GetConfig("worker:windowsDesiredCapacity")
		windowsMinSize, windowsMinSizeOK := ctx.GetConfig("worker:windowsMinSize")
		windowsMaxSize, windowsMaxSizeOK := ctx.GetConfig("worker:windowsMaxSize")

		_, windowsPasswordOK := ctx.GetConfig("worker:windowsPassword")
		err := validate(regionOK, windowsInstanceTypeOK, linuxInstanceTypeOK, windowsAMIOK, adminOK, accOK, linuxDesiredCapacityOK, linuxMinSizeOK, linuxMaxSizeOK, windowsDesiredCapacityOK, windowsMinSizeOK, windowsMaxSizeOK, windowsPasswordOK)
		if err != nil {
			return err
		}
		region = rg
		stackName = ctx.Stack()
		windowsDesiredCapacityInt, _ := strconv.ParseInt(windowsDesiredCapacity, 10, 64)
		windowsMinSizeInt, _ := strconv.ParseInt(windowsMinSize, 10, 64)
		windowsMaxSizeInt, _ := strconv.ParseInt(windowsMaxSize, 10, 64)

		linuxDesiredCapacityInt, _ := strconv.ParseInt(linuxDesiredCapacity, 10, 64)
		linuxMinSizeInt, _ := strconv.ParseInt(linuxMinSize, 10, 64)
		linuxMaxSizeInt, _ := strconv.ParseInt(linuxMaxSize, 10, 64)

		ami, err := lookupAMI(ctx, windowsAMI)
		if err != nil {
			return err
		}
		network := new(Network)
		err = setupEKSNetwork(ctx, network)
		if err != nil {
			return err
		}
		clusterRole, err := createClusterRole(ctx, getStackNameRegional("ClusterRole"))
		if err != nil {
			return err
		}
		systemRole, err := createWorkerRole(ctx, getStackNameRegional("SystemRole", "WorkloadCluster"))
		if err != nil {
			return err
		}
		winWorkerRole, err := createWorkerRole(ctx, getStackNameRegional("WindowsWorkerRole", "WorkloadCluster"))
		if err != nil {
			return err
		}
		linuxWorkerRole, err := createWorkerRole(ctx, getStackNameRegional("LinuxWorkerRole", "WorkloadCluster"))
		if err != nil {
			return err
		}
		workloadWorkerSecurityGroup, err := createWorkerSecurityGroup(ctx, network.Vpc)
		if err != nil {
			return err
		}
		internalId, err := allowFromSecurityGroup(ctx, workloadWorkerSecurityGroup, workloadWorkerSecurityGroup, "worker-sg", "worker-sg")
		if err != nil {
			return err
		}
		ctx.Export("InternalSecurityGroupID", internalId)
		clusterInstanceProfile, err := iam.NewInstanceProfile(ctx, getStackNameRegional("ClusterInstanceProfile"), &iam.InstanceProfileArgs{
			Role: clusterRole.Name,
		}, pulumi.DependsOn([]pulumi.Resource{clusterRole}))
		if err != nil {
			return err
		}
		workloadCluster, err := eks.NewCluster(ctx, getStackNameRegional("WorkloadCluster"), &eks.ClusterArgs{
			CreateOidcProvider: pulumi.BoolPtr(true),
			InstanceRoles: iam.RoleArray{
				linuxWorkerRole,
				winWorkerRole,
				systemRole,
			},
			Name:                         pulumi.String(getStackNameRegional("WorkloadCluster")),
			NodeAssociatePublicIpAddress: falsePtr,
			PrivateSubnetIds:             network.getPrivateSubnetIds(),
			ProviderCredentialOpts:       eks.KubeconfigOptionsArgs{},
			PublicSubnetIds:              network.getPublicSubnetIds(),
			RoleMappings: eks.RoleMappingArray{
				&eks.RoleMappingArgs{
					Groups:   pulumi.StringArray{pulumi.String("system:bootstrappers"), pulumi.String("system:nodes"), pulumi.String("eks:kube-proxy-windows")},
					RoleArn:  winWorkerRole.Arn,
					Username: pulumi.String("system:node:{{EC2PrivateDNSName}}"),
				},
			},
			ServiceRole:          clusterRole,
			SkipDefaultNodeGroup: truePtr,
			Tags: pulumi.StringMap{
				"ClusterName": pulumi.String(getStackNameRegional("WorkloadCluster")),
			},
			UseDefaultVpcCni: truePtr,
			ClusterSecurityGroupTags: pulumi.StringMap{
				"ClusterName": pulumi.String(getStackNameRegional("ClusterSecurityGroup", "WorkloadCluster")),
				"Type":        pulumi.String("ManagedSecurityGroup"),
				"For":         pulumi.String("Cluster"),
			},
			NodeSecurityGroupTags: pulumi.StringMap{
				"ClusterName": pulumi.String(getStackNameRegional("WorkerSecurityGroup", "WorkloadCluster")),
				"Type":        pulumi.String("ManagedSecurityGroup"),
				"For":         pulumi.String("Node"),
			},
			UserMappings: eks.UserMappingArray{
				&eks.UserMappingArgs{
					Groups:   pulumi.StringArray{pulumi.String("system:masters")},
					Username: pulumi.String(adminUsername),
					UserArn:  pulumi.String("arn:aws:iam::" + accountId + ":user/" + adminUsername),
				},
			},
			Version: pulumi.String(K8S_VERSION),
			VpcId:   network.Vpc.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{network.Vpc, clusterRole, clusterInstanceProfile, systemRole, linuxWorkerRole, winWorkerRole}))
		if err != nil {
			return err
		}
		result := pulumi.All(workloadCluster.NodeSecurityGroup, workloadCluster.ClusterSecurityGroup).ApplyT(func(args []interface{}) (interface{}, error) {
			nodeSecurityGroup := args[0].(*ec2.SecurityGroup)
			clusterSecurityGroup := args[1].(*ec2.SecurityGroup)
			fmt.Println("Applying Worker Security Group Rules [FROM NODE]")
			wn, err := allowFromSecurityGroup(ctx, workloadWorkerSecurityGroup, nodeSecurityGroup, "worker-sg", "node-sg")
			if err != nil {
				return nil, err
			}
			fmt.Println("Applying Node Security Group Rules [FROM WORKER]")
			nw, err := allowFromSecurityGroup(ctx, nodeSecurityGroup, workloadWorkerSecurityGroup, "node-sg", "worker-sg")
			if err != nil {
				return nil, err
			}
			fmt.Println("Applying WORKER Security Group Rules [FROM CLUSTER]")
			ws, err := allowFromSecurityGroup(ctx, workloadWorkerSecurityGroup, clusterSecurityGroup, "worker-sg", "cluster-sg")
			if err != nil {
				return nil, err
			}
			fmt.Println("Applying cluster Security Group Rules [FROM WORKER]")
			cw, err := allowFromSecurityGroup(ctx, clusterSecurityGroup, workloadWorkerSecurityGroup, "cluster-sg", "worker-sg")
			if err != nil {
				return nil, err
			}

			return []interface{}{
				wn, nw, ws, cw,
			}, nil
		})
		ctx.Export("SecurityGroupRules", result)
		systemLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("SystemLaunchTemplate", "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/sda1"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize:          pulumi.Int(100),
						VolumeType:          pulumi.String("gp3"),
						DeleteOnTermination: pulumi.String("true"),
					},
				},
			},
			Name: pulumi.String(getStackNameRegional("SystemLaunchTemplate", "WorkloadCluster")),
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("SystemLaunchTemplate", "WorkloadCluster")),
					},
				},
			},
			VpcSecurityGroupIds: pulumi.StringArray{
				workloadWorkerSecurityGroup.ID(),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadWorkerSecurityGroup, workloadCluster}))
		systemNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("SystemNodeGroup", "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("SystemNodeGroup", "WorkloadCluster")),
			ClusterName:   workloadCluster.EksCluster.Name(),
			NodeRoleArn:   systemRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(3),
				MinSize:     pulumi.Int(1),
			},
			SubnetIds: network.getPrivateSubnetIds(),
			InstanceTypes: pulumi.StringArray{
				pulumi.String("t3.medium"),
			},
			LaunchTemplate: &awsEKS.NodeGroupLaunchTemplateArgs{
				Id:      systemLaunchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			Labels: pulumi.StringMap{
				"type": pulumi.String("system"),
			},
			Tags: pulumi.StringMap{
				"Name":     pulumi.String(getStackNameRegional("SystemNodeGroup", "WorkloadCluster")),
				"workload": pulumi.String("system"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, systemRole, systemLaunchTemplate}))
		kubeconfig := workloadCluster.Kubeconfig.ApplyT(func(kc interface{}) (string, error) {
			content := kc.(map[string]interface{})
			bytes, err := json.Marshal(content)
			if err != nil {
				return "", errors.New("failed to marshal kubeconfig")
			}
			_ = os.WriteFile("kubeconfig", bytes, 0644)
			return string(bytes), nil
		}).(pulumi.StringOutput)
		k8sProvider, err := kubernetes.NewProvider(ctx, "k8sProvider", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, systemNodeGroup}))
		if err != nil {
			return err
		}
		kubeDns, err := corev1.GetService(ctx, "kube-system/kube-dns", pulumi.ID("kube-system/kube-dns"), nil, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}
		linuxLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("LinuxLaunchTemplate", "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/sda1"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize: pulumi.Int(100),
					},
				},
			},
			VpcSecurityGroupIds: pulumi.StringArray{
				workloadWorkerSecurityGroup.ID(),
			},
			Name: pulumi.String(getStackNameRegional("LinuxLaunchTemplate", "WorkloadCluster")),
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("LinuxLaunchTemplate", "WorkloadCluster")),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadWorkerSecurityGroup, workloadCluster}))
		linuxNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("LinuxNodeGroup", "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("LinuxNodeGroup", "WorkloadCluster")),
			ClusterName:   workloadCluster.EksCluster.Name(),
			NodeRoleArn:   linuxWorkerRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(linuxDesiredCapacityInt),
				MaxSize:     pulumi.Int(linuxMaxSizeInt),
				MinSize:     pulumi.Int(linuxMinSizeInt),
			},
			AmiType:   pulumi.String("AL2_x86_64_GPU"),
			SubnetIds: network.getPublicSubnetIds(),
			InstanceTypes: pulumi.StringArray{
				pulumi.String(linuxInstanceType),
			},
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
				"Name": pulumi.String(getStackNameRegional("LinuxNodeGroup", "WorkloadCluster")),
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, linuxWorkerRole, linuxLaunchTemplate}))
		windowsLaunchTemplate, err := ec2.NewLaunchTemplate(ctx, getStackNameRegional("WindowsLaunchTemplate", "WorkloadCluster"), &ec2.LaunchTemplateArgs{
			Name:         pulumi.String(getStackNameRegional("WindowsLaunchTemplate", "WorkloadCluster")),
			ImageId:      pulumi.String(ami.ImageId),
			InstanceType: pulumi.String(windowsInstanceType),
			VpcSecurityGroupIds: pulumi.StringArray{
				workloadWorkerSecurityGroup.ID(),
			},
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/sda1"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize:          pulumi.Int(150),
						VolumeType:          pulumi.String("gp3"),
						DeleteOnTermination: pulumi.String("true"),
					},
				},
			},
			UserData: getWindowsUserData(ctx, workloadCluster, kubeDns.Spec.ClusterIP()),

			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(getStackNameRegional("WindowsLaunchTemplate", "WorkloadCluster")),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{workloadWorkerSecurityGroup, workloadCluster, kubeDns, systemNodeGroup}))
		windowsNodeGroup, err := awsEKS.NewNodeGroup(ctx, getStackNameRegional("WindowsNodeGroup", "WorkloadCluster"), &awsEKS.NodeGroupArgs{
			NodeGroupName: pulumi.String(getStackNameRegional("WindowsNodeGroup", "WorkloadCluster")),
			ClusterName:   workloadCluster.EksCluster.Name(),
			NodeRoleArn:   winWorkerRole.Arn,
			ScalingConfig: &awsEKS.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(windowsDesiredCapacityInt),
				MaxSize:     pulumi.Int(windowsMaxSizeInt),
				MinSize:     pulumi.Int(windowsMinSizeInt),
			},
			SubnetIds: network.getPublicSubnetIds(),
			LaunchTemplate: &awsEKS.NodeGroupLaunchTemplateArgs{
				Id:      windowsLaunchTemplate.ID(),
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
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, winWorkerRole, windowsLaunchTemplate, systemNodeGroup}))

		clusterAutoscalerPolicy, err := iam.NewPolicy(ctx, getStackNameRegional("AutoScalerPolicy", "WorkloadCluster"), &iam.PolicyArgs{
			Description: pulumi.String("Allows the cluster autoscaler to access AWS resources"),
			Name:        pulumi.String(getStackNameRegional("AutoScalerPolicy", "WorkloadCluster")),
			Policy:      pulumi.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["autoscaling:DescribeAutoScalingGroups","autoscaling:DescribeAutoScalingInstances","autoscaling:DescribeLaunchConfigurations","autoscaling:DescribeScalingActivities","autoscaling:DescribeTags","ec2:DescribeImages","ec2:DescribeInstanceTypes","ec2:DescribeLaunchTemplateVersions","ec2:GetInstanceTypesFromInstanceRequirements","eks:DescribeNodegroup"],"Resource":["*"]},{"Effect":"Allow","Action":["autoscaling:SetDesiredCapacity","autoscaling:TerminateInstanceInAutoScalingGroup"],"Resource":["*"]}]}`),
		}, pulumi.DependsOn([]pulumi.Resource{workloadCluster}))
		if err != nil {
			return err
		}
		// Create Role for Cluster Autoscaler

		workloadCluster.Core.OidcProvider().Arn().ApplyT(func(arn interface{}) (interface{}, error) {
			role := `{"Version":"2012-10-17","Statement":[{"Sid":"","Effect":"Allow","Principal":{"Federated":"` + arn.(string) + `"},"Action":"sts:AssumeRoleWithWebIdentity"}]}`
			createdRole, err := iam.NewRole(ctx, getStackNameRegional("AutoScalerRole", "WorkloadCluster"), &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(role),
				Description:      pulumi.String("Allows the cluster autoscaler to access AWS resources"),
				Name:             pulumi.String(getStackNameRegional("AutoScalerRole", "WorkloadCluster")),
			}, pulumi.DependsOn([]pulumi.Resource{workloadCluster, clusterAutoscalerPolicy}))
			if err != nil {
				return nil, err
			}
			policyAttachment, err := iam.NewPolicyAttachment(ctx, getStackNameRegional("AutoScalerPolicyAttachment", "WorkloadCluster"), &iam.PolicyAttachmentArgs{
				PolicyArn: clusterAutoscalerPolicy.Arn,
				Roles:     pulumi.Array{createdRole.Name},
			}, pulumi.DependsOn([]pulumi.Resource{clusterAutoscalerPolicy, createdRole}))
			if err != nil {
				return nil, err
			}
			serviceAccount, err := corev1.NewServiceAccount(ctx, "cluster-autoscaler", &corev1.ServiceAccountArgs{
				Metadata: metav1.ObjectMetaArgs{
					Name:      pulumi.String("cluster-autoscaler"),
					Namespace: pulumi.String("kube-system"),
					Labels: pulumi.StringMap{
						"app.kubernetes.io/name": pulumi.String("cluster-autoscaler"),
					},
					Annotations: pulumi.StringMap{
						"eks.amazonaws.com/role-arn": createdRole.Arn,
					},
				},
			}, pulumi.Provider(k8sProvider))
			if err != nil {
				return nil, err
			}
			// Create Cluster AutoScaler
			release, err := helm.NewRelease(ctx, "cluster-autoscaler", &helm.ReleaseArgs{
				Namespace: pulumi.String("kube-system"),
				Name:      pulumi.String("cluster-autoscaler"),
				RepositoryOpts: helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://kubernetes.github.io/autoscaler"),
				},
				Chart:   pulumi.String("cluster-autoscaler"),
				Version: pulumi.String("9.36.0"),
				Values: pulumi.Map{
					"cloudProvider": pulumi.String("aws"),
					"awsRegion":     pulumi.String(region),
					"autoDiscovery": pulumi.Map{
						"clusterName": workloadCluster.EksCluster.Name(),
					},
					"rbac": pulumi.Map{
						"create": pulumi.Bool(true),
						"serviceAccount": pulumi.Map{
							"name":   pulumi.String("cluster-autoscaler"),
							"create": pulumi.Bool(false),
						},
					},
				},
				WaitForJobs: pulumi.Bool(true),
			}, pulumi.Provider(k8sProvider), pulumi.DependsOn([]pulumi.Resource{workloadCluster, serviceAccount, policyAttachment}))
			return release, err
		})

		ctx.Export("SystemNodeGroup", systemNodeGroup.ID())
		ctx.Export("LinuxNodeGroup", linuxNodeGroup.ID())
		ctx.Export("WindowsNodeGroup", windowsNodeGroup.ID())
		ctx.Export("WorkerSecurityGroup", workloadWorkerSecurityGroup.ID())
		ctx.Export("ClusterCoreSecurityGroup", workloadCluster.Core.ClusterSecurityGroup().ApplyT(func(sg interface{}) (pulumi.IDOutput, error) {
			return sg.(*ec2.SecurityGroup).ID(), nil
		}).(pulumi.IDOutput))
		ctx.Export("ClusterSecurityGroup", workloadCluster.ClusterSecurityGroup.ApplyT(func(sg interface{}) (pulumi.IDOutput, error) {
			return sg.(*ec2.SecurityGroup).ID(), nil
		}).(pulumi.IDOutput))
		ctx.Export("ClusterNodeSecurityGroup", workloadCluster.NodeSecurityGroup.ApplyT(func(sg interface{}) (pulumi.IDOutput, error) {
			return sg.(*ec2.SecurityGroup).ID(), nil
		}).(pulumi.IDOutput))

		//ctx.Export("EKSCluster", workloadCluster.Kubeconfig)
		//ctx.Export("WindowsUserData", getWindowsUserData(ctx, workloadCluster, kubeDns.Spec.ClusterIP()))
		return nil
	})
}
