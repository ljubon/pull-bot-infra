package main

import (
	// "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	awsxLb "github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const PULL_CONTAINER = "ghcr.io/ljubon/pull/pull:latest"
const BUCKET = "arn:aws:s3:::pullbot-envs/.env"
const PRIVATE_KEY = "arn:aws:secretsmanager:us-east-1:341894770476:secret:PULL_PRIVATE_KEY-dZhI2J"
const TASK_ROLE = "arn:aws:iam::341894770476:role/ecsTaskExecutionRole"
const VPC_ID = "vpc-0fbca88fc6fab7a0f"
const SECURITY_GROUP = "sg-01a8e31f04b83e53d"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		cluster, err := ecs.NewCluster(ctx, "pull-pulumi-cluster", nil)
		if err != nil {
			return err
		}
		loadBalancer, err := awsxLb.NewApplicationLoadBalancer(ctx, "lb", &awsxLb.ApplicationLoadBalancerArgs {
			DefaultTargetGroupPort: pulumi.Int(3000),
		})
		if err != nil {
			return err
		}
		_, err = awsxEcs.NewEC2Service(ctx, "pull-pulumi-service", &awsxEcs.EC2ServiceArgs{
			Cluster:      cluster.Arn,
			DesiredCount: pulumi.Int(5),
			NetworkConfiguration: ecs.ServiceNetworkConfigurationArgs{
				SecurityGroups: pulumi.StringArray{
					pulumi.String(SECURITY_GROUP),
				},
				Subnets: pulumi.StringArray{
					pulumi.String("subnet-02c6606e6327a2524"),
					pulumi.String("subnet-0e8a610a8547bd5a1"),
					pulumi.String("subnet-026cf2674f7b9e008"),
					pulumi.String("subnet-06999ccd7f8a4d538"),
					pulumi.String("subnet-0a278c1c001e0608e"),
					pulumi.String("subnet-0ac5a47ac46d2a3d8"),
				},
			},
			TaskDefinitionArgs: &awsxEcs.EC2ServiceTaskDefinitionArgs{
				Container: &awsxEcs.TaskDefinitionContainerDefinitionArgs{
					Image:     pulumi.String(PULL_CONTAINER),
					Cpu:       pulumi.Int(512),
					Memory:    pulumi.Int(512),
					Essential: pulumi.Bool(true),
					Environment: awsxEcs.TaskDefinitionKeyValuePairArray{
						awsxEcs.TaskDefinitionKeyValuePairArgs{
							Name:  pulumi.String("PRIVATE_KEY"),
							Value: pulumi.String(PRIVATE_KEY),
						},
					},
					EnvironmentFiles: awsxEcs.TaskDefinitionEnvironmentFileArray{
						awsxEcs.TaskDefinitionEnvironmentFileArgs{
							Type:  pulumi.String("s3"),
							Value: pulumi.String(BUCKET),
						},
					},
					PortMappings: awsxEcs.TaskDefinitionPortMappingArray{
						awsxEcs.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.Int(3000),
							HostPort:      pulumi.Int(3000),
							Protocol: pulumi.String("tcp"),
							TargetGroup:   loadBalancer.DefaultTargetGroup,
						},
					},
				},
				TaskRole: &awsx.DefaultRoleWithPolicyArgs{
					RoleArn: pulumi.String(TASK_ROLE),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("url", loadBalancer.LoadBalancer.ApplyT(func(loadbal *lb.LoadBalancer) (string, error) {
			return loadbal.DnsName.ElementType().String(), nil
		}).(pulumi.StringOutput))
		return nil
	})
}
