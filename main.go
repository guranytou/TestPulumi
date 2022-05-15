package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := ec2.NewVpc(ctx, "test-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("192.168.0.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("test-vpc"),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("vpc", vpc.ID())
		return nil
	})
}
