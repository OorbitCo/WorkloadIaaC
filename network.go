package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Network struct {
	Vpc            *ec2.Vpc
	PublicSubnets  []*ec2.Subnet
	PrivateSubnets []*ec2.Subnet
}

func (n *Network) getSubnetIds() pulumi.StringArray {
	subnets := pulumi.StringArray{}
	for _, subnet := range n.PublicSubnets {
		subnets = append(subnets, subnet.ID())
	}
	for _, subnet := range n.PrivateSubnets {
		subnets = append(subnets, subnet.ID())
	}
	return subnets
}
func (n *Network) getPublicSubnetIds() pulumi.StringArray {
	subnets := pulumi.StringArray{}
	for _, subnet := range n.PublicSubnets {
		subnets = append(subnets, subnet.ID())
	}
	return subnets
}
func (n *Network) getPrivateSubnetIds() pulumi.StringArray {
	subnets := pulumi.StringArray{}
	for _, subnet := range n.PrivateSubnets {
		subnets = append(subnets, subnet.ID())
	}
	return subnets
}
