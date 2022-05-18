package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := ec2.NewVpc(ctx, "example_vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.100.0.0/16"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_vpc"),
			},
		})
		if err != nil {
			return err
		}

		pubSub1a, err := ec2.NewSubnet(ctx, "pubSub1a", &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.100.0.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1a"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_public_1a"),
			},
		})

		pubSub1c, err := ec2.NewSubnet(ctx, "pubSub1c", &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.100.1.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1c"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_public_1c"),
			},
		})

		priSub1a, err := ec2.NewSubnet(ctx, "priSub1a", &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.100.100.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1a"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_private_1a"),
			},
		})

		eip, err := ec2.NewEip(ctx, "eip", &ec2.EipArgs{
			Vpc: pulumi.Bool(true),
		})

		igw, err := ec2.NewInternetGateway(ctx, "igw", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_igw"),
			},
		})

		natgw, err := ec2.NewNatGateway(ctx, "natgw", &ec2.NatGatewayArgs{
			AllocationId: eip.ID(),
			SubnetId:     pubSub1a.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{eip}))

		pubRoutaTable, err := ec2.NewRouteTable(ctx, "pubRouteTable", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_route_table_public"),
			},
		})

		_, err = ec2.NewRoute(ctx, "pubRoute", &ec2.RouteArgs{
			RouteTableId:         pubRoutaTable.ID(),
			DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
			GatewayId:            igw.ID(),
		})

		_, err = ec2.NewRouteTableAssociation(ctx, "pubRoute1a", &ec2.RouteTableAssociationArgs{
			SubnetId:     pubSub1a.ID(),
			RouteTableId: pubRoutaTable.ID(),
		})

		_, err = ec2.NewRouteTableAssociation(ctx, "pubRoute1c", &ec2.RouteTableAssociationArgs{
			SubnetId:     pubSub1c.ID(),
			RouteTableId: pubRoutaTable.ID(),
		})

		priRouteTable, err := ec2.NewRouteTable(ctx, "priRouteTable", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example_route_table_private"),
			},
		})

		_, err = ec2.NewRoute(ctx, "priRoute", &ec2.RouteArgs{
			RouteTableId:         priRouteTable.ID(),
			DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
			NatGatewayId:         natgw.ID(),
		})

		_, err = ec2.NewRouteTableAssociation(ctx, "priRoute1a", &ec2.RouteTableAssociationArgs{
			SubnetId:     priSub1a.ID(),
			RouteTableId: priRouteTable.ID(),
		})

		sgForALB, err := ec2.NewSecurityGroup(ctx, "sgForALB", &ec2.SecurityGroupArgs{
			Name:  pulumi.String("example_sg_for_ALB"),
			VpcId: vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					FromPort: pulumi.Int(80),
					ToPort:   pulumi.Int(80),
					Protocol: pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
		})

		alb, err := lb.NewLoadBalancer(ctx, "ALB", &lb.LoadBalancerArgs{
			Name:             pulumi.String("example"),
			LoadBalancerType: pulumi.String("application"),
			SecurityGroups: pulumi.StringArray{
				sgForALB.ID(),
			},
			Subnets: pulumi.StringArray{
				pubSub1a.ID(),
				pubSub1c.ID(),
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("example"),
			},
		})

		httpTG, err := lb.NewTargetGroup(ctx, "httpTG", &lb.TargetGroupArgs{
			Name:     pulumi.String("HTTPTG"),
			Port:     pulumi.Int(80),
			Protocol: pulumi.String("HTTP"),
			VpcId:    vpc.ID(),
			HealthCheck: lb.TargetGroupHealthCheckArgs{
				Path:     pulumi.String("/"),
				Matcher:  pulumi.String("403"),
				Port:     pulumi.String("80"),
				Protocol: pulumi.String("HTTP"),
			},
		})

		_, err = lb.NewListener(ctx, "listener", &lb.ListenerArgs{
			LoadBalancerArn: alb.Arn,
			Port:            pulumi.Int(80),
			Protocol:        pulumi.String("HTTP"),
			DefaultActions: lb.ListenerDefaultActionArray{
				&lb.ListenerDefaultActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: httpTG.Arn,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{alb}))

		sgForInstance, err := ec2.NewSecurityGroup(ctx, "sgForInstance", &ec2.SecurityGroupArgs{
			Name:  pulumi.String("example_sg_for_instance"),
			VpcId: vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					FromPort: pulumi.Int(80),
					ToPort:   pulumi.Int(80),
					Protocol: pulumi.String("tcp"),
					SecurityGroups: pulumi.StringArray{
						sgForALB.ID(),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
		})

		f, err := os.Open("install_apache.sh")
		b, err := ioutil.ReadAll(f)
		enc := base64.StdEncoding.EncodeToString(b)

		ins, err := ec2.NewInstance(ctx, "instance", &ec2.InstanceArgs{
			Ami:          pulumi.String("ami-0992fc94ca0f1415a"),
			InstanceType: pulumi.String("t2.micro"),
			SubnetId:     priSub1a.ID(),
			VpcSecurityGroupIds: pulumi.StringArray{
				sgForInstance.ID(),
			},
			UserDataBase64: pulumi.String(enc),
		})

		_, err = lb.NewTargetGroupAttachment(ctx, "TGattach", &lb.TargetGroupAttachmentArgs{
			TargetGroupArn: httpTG.Arn,
			TargetId:       ins.ID(),
			Port:           pulumi.Int(80),
		})

		return nil
	})
}
