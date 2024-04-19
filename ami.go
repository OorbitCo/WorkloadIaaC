package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func lookupAMI(ctx *pulumi.Context, search string) (*ec2.LookupAmiResult, error) {
	ami, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		Filters: []ec2.GetAmiFilter{
			{
				Name: "product-code",
				Values: []string{
					"6c2ls17bo706uvbzvvx39aimt",
				},
			},
		},
	})
	return ami, err
}
