package main

import (
	"errors"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Ref : https://s3.us-west-2.amazonaws.com/amazon-eks/cloudformation/2020-10-29/amazon-eks-vpc-private-subnets.yaml
func setupEKSNetwork(ctx *pulumi.Context, network *Network) error {
	region, ok := ctx.GetConfig("aws:region")
	if !ok {
		return errors.New("missing required configuration 'aws:region'")
	}

	/*
		Parameters:

		  VpcBlock:
		    Type: String
		    Default: 192.168.0.0/16
		    Description: The CIDR range for the VPC. This should be a valid private (RFC 1918) CIDR range.

		  PublicSubnet01Block:
		    Type: String
		    Default: 192.168.0.0/18
		    Description: CidrBlock for public subnet 01 within the VPC

		  PublicSubnet02Block:
		    Type: String
		    Default: 192.168.64.0/18
		    Description: CidrBlock for public subnet 02 within the VPC

		  PrivateSubnet01Block:
		    Type: String
		    Default: 192.168.128.0/18
		    Description: CidrBlock for private subnet 01 within the VPC

		  PrivateSubnet02Block:
		    Type: String
		    Default: 192.168.192.0/18
		    Description: CidrBlock for private subnet 02 within the VPC
	*/
	type VpcParams struct {
		VpcBlock             string
		PublicSubnet01Block  string
		PublicSubnet02Block  string
		PrivateSubnet01Block string
		PrivateSubnet02Block string
	}
	params := VpcParams{
		VpcBlock:             "192.168.0.0/16",
		PublicSubnet01Block:  "192.168.0.0/18",
		PublicSubnet02Block:  "192.168.64.0/18",
		PrivateSubnet01Block: "192.168.128.0/18",
		PrivateSubnet02Block: "192.168.192.0/18",
	}
	/*
		Resources:
		  VPC:
		    Type: AWS::EC2::VPC
		    Properties:
		      CidrBlock:  !Ref VpcBlock
		      EnableDnsSupport: true
		      EnableDnsHostnames: true
		      Tags:
		      - Key: Name
		        Value: !Sub '${AWS::StackName}-VPC'

		  InternetGateway:
		    Type: "AWS::EC2::InternetGateway"

		  VPCGatewayAttachment:
		    Type: "AWS::EC2::VPCGatewayAttachment"
		    Properties:
		      InternetGatewayId: !Ref InternetGateway
		      VpcId: !Ref VPC

		  PublicRouteTable:
		    Type: AWS::EC2::RouteTable
		    Properties:
		      VpcId: !Ref VPC
		      Tags:
		      - Key: Name
		        Value: Public Subnets
		      - Key: Network
		        Value: Public

		  PrivateRouteTable01:
		    Type: AWS::EC2::RouteTable
		    Properties:
		      VpcId: !Ref VPC
		      Tags:
		      - Key: Name
		        Value: Private Subnet AZ1
		      - Key: Network
		        Value: Private01

		  PrivateRouteTable02:
		    Type: AWS::EC2::RouteTable
		    Properties:
		      VpcId: !Ref VPC
		      Tags:
		      - Key: Name
		        Value: Private Subnet AZ2
		      - Key: Network
		        Value: Private02

		  PublicRoute:
		    DependsOn: VPCGatewayAttachment
		    Type: AWS::EC2::Route
		    Properties:
		      RouteTableId: !Ref PublicRouteTable
		      DestinationCidrBlock: 0.0.0.0/0
		      GatewayId: !Ref InternetGateway

		  PrivateRoute01:
		    DependsOn:
		    - VPCGatewayAttachment
		    - NatGateway01
		    Type: AWS::EC2::Route
		    Properties:
		      RouteTableId: !Ref PrivateRouteTable01
		      DestinationCidrBlock: 0.0.0.0/0
		      NatGatewayId: !Ref NatGateway01

		  PrivateRoute02:
		    DependsOn:
		    - VPCGatewayAttachment
		    - NatGateway02
		    Type: AWS::EC2::Route
		    Properties:
		      RouteTableId: !Ref PrivateRouteTable02
		      DestinationCidrBlock: 0.0.0.0/0
		      NatGatewayId: !Ref NatGateway02

		  NatGateway01:
		    DependsOn:
		    - NatGatewayEIP1
		    - PublicSubnet01
		    - VPCGatewayAttachment
		    Type: AWS::EC2::NatGateway
		    Properties:
		      AllocationId: !GetAtt 'NatGatewayEIP1.AllocationId'
		      SubnetId: !Ref PublicSubnet01
		      Tags:
		      - Key: Name
		        Value: !Sub '${AWS::StackName}-NatGatewayAZ1'

		  NatGateway02:
		    DependsOn:
		    - NatGatewayEIP2
		    - PublicSubnet02
		    - VPCGatewayAttachment
		    Type: AWS::EC2::NatGateway
		    Properties:
		      AllocationId: !GetAtt 'NatGatewayEIP2.AllocationId'
		      SubnetId: !Ref PublicSubnet02
		      Tags:
		      - Key: Name
		        Value: !Sub '${AWS::StackName}-NatGatewayAZ2'

		  NatGatewayEIP1:
		    DependsOn:
		    - VPCGatewayAttachment
		    Type: 'AWS::EC2::EIP'
		    Properties:
		      Domain: vpc

		  NatGatewayEIP2:
		    DependsOn:
		    - VPCGatewayAttachment
		    Type: 'AWS::EC2::EIP'
		    Properties:
		      Domain: vpc

		  PublicSubnet01:
		    Type: AWS::EC2::Subnet
		    Metadata:
		      Comment: Subnet 01
		    Properties:
		      MapPublicIpOnLaunch: true
		      AvailabilityZone:
		        Fn::Select:
		        - '0'
		        - Fn::GetAZs:
		            Ref: AWS::Region
		      CidrBlock:
		        Ref: PublicSubnet01Block
		      VpcId:
		        Ref: VPC
		      Tags:
		      - Key: Name
		        Value: !Sub "${AWS::StackName}-PublicSubnet01"
		      - Key: kubernetes.io/role/elb
		        Value: 1

		  PublicSubnet02:
		    Type: AWS::EC2::Subnet
		    Metadata:
		      Comment: Subnet 02
		    Properties:
		      MapPublicIpOnLaunch: true
		      AvailabilityZone:
		        Fn::Select:
		        - '1'
		        - Fn::GetAZs:
		            Ref: AWS::Region
		      CidrBlock:
		        Ref: PublicSubnet02Block
		      VpcId:
		        Ref: VPC
		      Tags:
		      - Key: Name
		        Value: !Sub "${AWS::StackName}-PublicSubnet02"
		      - Key: kubernetes.io/role/elb
		        Value: 1

		  PrivateSubnet01:
		    Type: AWS::EC2::Subnet
		    Metadata:
		      Comment: Subnet 03
		    Properties:
		      AvailabilityZone:
		        Fn::Select:
		        - '0'
		        - Fn::GetAZs:
		            Ref: AWS::Region
		      CidrBlock:
		        Ref: PrivateSubnet01Block
		      VpcId:
		        Ref: VPC
		      Tags:
		      - Key: Name
		        Value: !Sub "${AWS::StackName}-PrivateSubnet01"
		      - Key: kubernetes.io/role/internal-elb
		        Value: 1

		  PrivateSubnet02:
		    Type: AWS::EC2::Subnet
		    Metadata:
		      Comment: Private Subnet 02
		    Properties:
		      AvailabilityZone:
		        Fn::Select:
		        - '1'
		        - Fn::GetAZs:
		            Ref: AWS::Region
		      CidrBlock:
		        Ref: PrivateSubnet02Block
		      VpcId:
		        Ref: VPC
		      Tags:
		      - Key: Name
		        Value: !Sub "${AWS::StackName}-PrivateSubnet02"
		      - Key: kubernetes.io/role/internal-elb
		        Value: 1

		  PublicSubnet01RouteTableAssociation:
		    Type: AWS::EC2::SubnetRouteTableAssociation
		    Properties:
		      SubnetId: !Ref PublicSubnet01
		      RouteTableId: !Ref PublicRouteTable

		  PublicSubnet02RouteTableAssociation:
		    Type: AWS::EC2::SubnetRouteTableAssociation
		    Properties:
		      SubnetId: !Ref PublicSubnet02
		      RouteTableId: !Ref PublicRouteTable

		  PrivateSubnet01RouteTableAssociation:
		    Type: AWS::EC2::SubnetRouteTableAssociation
		    Properties:
		      SubnetId: !Ref PrivateSubnet01
		      RouteTableId: !Ref PrivateRouteTable01

		  PrivateSubnet02RouteTableAssociation:
		    Type: AWS::EC2::SubnetRouteTableAssociation
		    Properties:
		      SubnetId: !Ref PrivateSubnet02
		      RouteTableId: !Ref PrivateRouteTable02

		  ControlPlaneSecurityGroup:
		    Type: AWS::EC2::SecurityGroup
		    Properties:
		      GroupDescription: Cluster communication with worker nodes
		      VpcId: !Ref VPC
	*/
	// Create a VPC
	VPC, err := ec2.NewVpc(ctx, getStackName("VPC", ctx.Stack()), &ec2.VpcArgs{
		CidrBlock:          pulumi.String(params.VpcBlock),
		EnableDnsSupport:   pulumi.Bool(true),
		EnableDnsHostnames: pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(getStackName("VPC", ctx.Stack())),
		},
	})
	if err != nil {
		return err
	}
	InternetGateway, err := ec2.NewInternetGateway(ctx, getStackName("IGW", ctx.Stack()), &ec2.InternetGatewayArgs{
		Tags: pulumi.StringMap{
			"Name": pulumi.String(getStackName("IGW", ctx.Stack())),
		},
	})
	if err != nil {
		return err
	}
	VPCGatewayAttachment, err := ec2.NewInternetGatewayAttachment(ctx, getStackName("VPCGatewayAttachment", ctx.Stack()), &ec2.InternetGatewayAttachmentArgs{
		InternetGatewayId: InternetGateway.ID(),
		VpcId:             VPC.ID(),
	})
	if err != nil {
		return err
	}
	PublicRouteTable, err := ec2.NewRouteTable(ctx, getStackName("PublicRouteTable", ctx.Stack()), &ec2.RouteTableArgs{
		VpcId: VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String("Public Subnets"),
			"Network": pulumi.String("Public"),
		},
	})
	if err != nil {
		return err
	}
	PrivateRouteTable01, err := ec2.NewRouteTable(ctx, getStackName("PrivateRouteTable01", ctx.Stack()), &ec2.RouteTableArgs{
		VpcId: VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String("Private Subnet AZ1"),
			"Network": pulumi.String("Private01"),
		},
	})
	if err != nil {
		return err
	}
	PrivateRouteTable02, err := ec2.NewRouteTable(ctx, getStackName("PrivateRouteTable02", ctx.Stack()), &ec2.RouteTableArgs{
		VpcId: VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String("Private Subnet AZ2"),
			"Network": pulumi.String("Private02"),
		},
	})
	if err != nil {
		return err
	}
	PublicSubnet01, err := ec2.NewSubnet(ctx, getStackName("PublicSubnet01", ctx.Stack()), &ec2.SubnetArgs{
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String(region + "a"),
		CidrBlock:           pulumi.String(params.PublicSubnet01Block),
		VpcId:               VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":                   pulumi.String(getStackName("PublicSubnet01", ctx.Stack())),
			"kubernetes.io/role/elb": pulumi.String("1"),
		},
	})
	if err != nil {
		return err
	}
	PublicSubnet02, err := ec2.NewSubnet(ctx, getStackName("PublicSubnet02", ctx.Stack()), &ec2.SubnetArgs{
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String(region + "b"),
		CidrBlock:           pulumi.String(params.PublicSubnet02Block),
		VpcId:               VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":                   pulumi.String(getStackName("PublicSubnet02", ctx.Stack())),
			"kubernetes.io/role/elb": pulumi.String("1"),
		},
	})
	if err != nil {
		return err
	}
	PrivateSubnet01, err := ec2.NewSubnet(ctx, getStackName("PrivateSubnet01", ctx.Stack()), &ec2.SubnetArgs{
		AvailabilityZone: pulumi.String(region + "a"),
		CidrBlock:        pulumi.String(params.PrivateSubnet01Block),
		VpcId:            VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":                            pulumi.String(getStackName("PrivateSubnet01", ctx.Stack())),
			"kubernetes.io/role/internal-elb": pulumi.String("1"),
		},
	})
	if err != nil {
		return err
	}
	PrivateSubnet02, err := ec2.NewSubnet(ctx, getStackName("PrivateSubnet02", ctx.Stack()), &ec2.SubnetArgs{
		AvailabilityZone: pulumi.String(region + "b"),
		CidrBlock:        pulumi.String(params.PrivateSubnet02Block),
		VpcId:            VPC.ID(),
		Tags: pulumi.StringMap{
			"Name":                            pulumi.String(getStackName("PrivateSubnet02", ctx.Stack())),
			"kubernetes.io/role/internal-elb": pulumi.String("1"),
		},
	})
	if err != nil {
		return err
	}
	PublicRoute, err := ec2.NewRoute(ctx, getStackName("PublicRoute", ctx.Stack()), &ec2.RouteArgs{
		RouteTableId:         PublicRouteTable.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		GatewayId:            InternetGateway.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment}))
	if err != nil {
		return err
	}
	NatGatewayEIP1, err := ec2.NewEip(ctx, getStackName("NatGatewayEIP1", ctx.Stack()), &ec2.EipArgs{
		Domain: pulumi.String("vpc"),
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment}))
	if err != nil {
		return err
	}
	NatGatewayEIP2, err := ec2.NewEip(ctx, getStackName("NatGatewayEIP2", ctx.Stack()), &ec2.EipArgs{
		Domain: pulumi.String("vpc"),
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment}))
	if err != nil {
		return err
	}
	NatGateway01, err := ec2.NewNatGateway(ctx, getStackName("NatGateway01", ctx.Stack()), &ec2.NatGatewayArgs{
		AllocationId: NatGatewayEIP1.ID(),
		SubnetId:     PublicSubnet01.ID(),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(getStackName("NatGateway01", ctx.Stack())),
		},
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment, PublicSubnet01, NatGatewayEIP1}))
	if err != nil {
		return err
	}
	NatGateway02, err := ec2.NewNatGateway(ctx, getStackName("NatGateway02", ctx.Stack()), &ec2.NatGatewayArgs{
		AllocationId: NatGatewayEIP2.ID(),
		SubnetId:     PublicSubnet02.ID(),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(getStackName("NatGateway02", ctx.Stack())),
		},
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment, PublicSubnet02, NatGatewayEIP2}))
	if err != nil {
		return err
	}
	PrivateRoute01, err := ec2.NewRoute(ctx, getStackName("PrivateRoute01", ctx.Stack()), &ec2.RouteArgs{
		RouteTableId:         PrivateRouteTable01.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		NatGatewayId:         NatGateway01.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment, NatGateway01}))
	if err != nil {
		return err
	}
	PrivateRoute02, err := ec2.NewRoute(ctx, getStackName("PrivateRoute02", ctx.Stack()), &ec2.RouteArgs{
		RouteTableId:         PrivateRouteTable02.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		NatGatewayId:         NatGateway02.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{VPCGatewayAttachment, NatGateway02}))
	if err != nil {
		return err
	}
	PublicRouteTableAssociation01, err := ec2.NewRouteTableAssociation(ctx, getStackName("PublicRouteTableAssociation01", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     PublicSubnet01.ID(),
		RouteTableId: PublicRouteTable.ID(),
	})
	if err != nil {
		return err
	}
	PublicRouteTableAssociation02, err := ec2.NewRouteTableAssociation(ctx, getStackName("PublicRouteTableAssociation02", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     PublicSubnet02.ID(),
		RouteTableId: PublicRouteTable.ID(),
	})
	if err != nil {
		return err
	}
	PrivateRouteTableAssociation01, err := ec2.NewRouteTableAssociation(ctx, getStackName("PrivateRouteTableAssociation01", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     PrivateSubnet01.ID(),
		RouteTableId: PrivateRouteTable01.ID(),
	})
	if err != nil {
		return err
	}
	PrivateRouteTableAssociation02, err := ec2.NewRouteTableAssociation(ctx, getStackName("PrivateRouteTableAssociation02", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     PrivateSubnet02.ID(),
		RouteTableId: PrivateRouteTable02.ID(),
	})
	if err != nil {
		return err
	}
	ClusterSecurityGroup, err := ec2.NewSecurityGroup(ctx, getStackName("ClusterSecurityGroup", ctx.Stack()), &ec2.SecurityGroupArgs{
		Name:        pulumi.String(getStackName("ClusterSecurityGroup", ctx.Stack())),
		Description: pulumi.String("Cluster communication with worker nodes"),
		VpcId:       VPC.ID(),
		Ingress:     ec2.SecurityGroupIngressArray{},
		Egress: ec2.SecurityGroupEgressArray{
			// Allow all outbound traffic
			&ec2.SecurityGroupEgressArgs{
				CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				Description:    pulumi.String("Allow all outbound traffic"),
				FromPort:       pulumi.Int(0),
				ToPort:         pulumi.Int(0),
				Protocol:       pulumi.String("-1"),
				Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{VPC}))
	if err != nil {
		return err
	}
	ctx.Export("VPC", VPC.ID())
	ctx.Export("InternetGateway", InternetGateway.ID())
	ctx.Export("VPCGatewayAttachment", VPCGatewayAttachment.ID())
	ctx.Export("PublicRouteTable", PublicRouteTable.ID())
	ctx.Export("PrivateRouteTable01", PrivateRouteTable01.ID())
	ctx.Export("PrivateRouteTable02", PrivateRouteTable02.ID())
	ctx.Export("PublicSubnet01", PublicSubnet01.ID())
	ctx.Export("PublicSubnet02", PublicSubnet02.ID())
	ctx.Export("PrivateSubnet01", PrivateSubnet01.ID())
	ctx.Export("PrivateSubnet02", PrivateSubnet02.ID())
	ctx.Export("NatGateway01", NatGateway01.ID())
	ctx.Export("NatGateway02", NatGateway02.ID())
	ctx.Export("NatGatewayEIP1", NatGatewayEIP1.ID())
	ctx.Export("NatGatewayEIP2", NatGatewayEIP2.ID())
	ctx.Export("PublicRoute", PublicRoute.ID())
	ctx.Export("PrivateRoute01", PrivateRoute01.ID())
	ctx.Export("PrivateRoute02", PrivateRoute02.ID())
	ctx.Export("PublicRouteTableAssociation01", PublicRouteTableAssociation01.ID())
	ctx.Export("PublicRouteTableAssociation02", PublicRouteTableAssociation02.ID())
	ctx.Export("PrivateRouteTableAssociation01", PrivateRouteTableAssociation01.ID())
	ctx.Export("PrivateRouteTableAssociation02", PrivateRouteTableAssociation02.ID())
	ctx.Export("ClusterSecurityGroup", ClusterSecurityGroup.ID())
	network.PublicSubnets = []*ec2.Subnet{PublicSubnet01, PublicSubnet02}
	network.PrivateSubnets = []*ec2.Subnet{PrivateSubnet01, PrivateSubnet02}
	network.Vpc = VPC
	network.ClusterSecurityGroup = ClusterSecurityGroup
	return nil
}
func createWorkerSecurityGroup(ctx *pulumi.Context, vpc *ec2.Vpc) (*ec2.SecurityGroup, error) {
	ingressSecurityGroupArgs := ec2.SecurityGroupIngressArray{
		// Allow all inbound traffic
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow exposed node ports"),
			FromPort:       pulumi.Int(30000),
			ToPort:         pulumi.Int(32767),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("tcp"),
		},
		// Allow Turn Server Port TCP
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow Turn Server Port"),
			FromPort:       pulumi.Int(3478),
			ToPort:         pulumi.Int(3478),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("tcp"),
		},
		// Allow Turn Server Port UDP
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow Turn Server Port"),
			FromPort:       pulumi.Int(3478),
			ToPort:         pulumi.Int(3478),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("udp"),
		},
		// Allow Web Server Port 80 TCP
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow Web Server Port"),
			FromPort:       pulumi.Int(80),
			ToPort:         pulumi.Int(80),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("tcp"),
		},
		// Allow Web Server Port 443 TCP
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow Web Server Port"),
			FromPort:       pulumi.Int(443),
			ToPort:         pulumi.Int(443),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("tcp"),
		},
		//RDP Debug Port TODO: Remove this
		&ec2.SecurityGroupIngressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("RDP DEBUG"),
			FromPort:       pulumi.Int(3389),
			ToPort:         pulumi.Int(3389),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
			Protocol:       pulumi.String("tcp"),
		},
	}
	egressSecurityGroupArgs := ec2.SecurityGroupEgressArray{
		&ec2.SecurityGroupEgressArgs{
			CidrBlocks:     pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			Description:    pulumi.String("Allow all outbound traffic"),
			FromPort:       pulumi.Int(0),
			ToPort:         pulumi.Int(0),
			Protocol:       pulumi.String("-1"),
			Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
		},
	}
	SecurityGroup, err := ec2.NewSecurityGroup(ctx, getStackName("WorkerSecurityGroup", ctx.Stack()), &ec2.SecurityGroupArgs{
		Name:        pulumi.String(getStackName("WorkerSecurityGroup", ctx.Stack())),
		Description: pulumi.String("Cluster communication with worker nodes"),
		Ingress:     ingressSecurityGroupArgs,
		Egress:      egressSecurityGroupArgs,
		VpcId:       vpc.ID(),
		Tags: pulumi.StringMap{
			"Description": pulumi.String(getStackName("WorkerSecurityGroup", ctx.Stack())),
		},
	}, pulumi.DependsOn([]pulumi.Resource{vpc}))
	return SecurityGroup, err
}
