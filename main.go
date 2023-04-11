package main

import (
	"encoding/base64"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/autoscaling"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"strconv"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	awsxLb "github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
)

const PULL_CONTAINER = "ghcr.io/ljubon/pull/pull:latest"
const BUCKET = "arn:aws:s3:::pullbot-envs/.env"
const PRIVATE_KEY_ARN = "arn:aws:secretsmanager:us-east-1:341894770476:secret:PULL_PRIVATE_KEY-dZhI2J"
const TASK_ROLE = "arn:aws:iam::341894770476:role/ecsTaskExecutionRole"
const VPC_ID = "vpc-0fbca88fc6fab7a0f"
const SECURITY_GROUP = "sg-01a8e31f04b83e53d"
const CLUSTER_NAME = "pull-pulumi-cluster"
const SERVICE_NAME = "pull-pulumi-service"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		encodedUserData := pulumi.All("pull-pulumi-cluster").ApplyT(func(args []interface{}) (string, error) {
			userData := "echo ECS_CLUSTER=pull-pulumi-cluster >> /etc/ecs/ecs.config"
			return base64.StdEncoding.EncodeToString([]byte(userData)), nil
		}).(pulumi.StringOutput)

		// Create an EC2 launch template
		launchTemplate, err := ec2.NewLaunchTemplate(ctx, "pull-pulumi-launch-template", &ec2.LaunchTemplateArgs{
			ImageId:      pulumi.String("ami-0c76be34ffbfb0b14"),
			InstanceType: pulumi.String("t2.small"),
			UserData:     encodedUserData,
			KeyName:      pulumi.String("pullbot"),
		})
		if err != nil {
			return err
		}

		latestVersion := launchTemplate.LatestVersion.ApplyT(func(latestVersion int) string {
			return strconv.Itoa(latestVersion)
		}).(pulumi.StringOutput)

		// Create an Auto Scaling group
		autoScalingGroup, err := autoscaling.NewGroup(ctx, "pull-pulumi-asg", &autoscaling.GroupArgs{
			AvailabilityZones: pulumi.StringArray{
				pulumi.String("us-east-1a"),
				pulumi.String("us-east-1b"),
			},
			LaunchTemplate: autoscaling.GroupLaunchTemplateArgs{
				Id:      launchTemplate.ID(),
				Version: latestVersion,
			},
			DesiredCapacity: pulumi.Int(1),
			MinSize:         pulumi.Int(1),
			MaxSize:         pulumi.Int(1),
		})
		if err != nil {
			return err
		}

		// Create the EC2 capacity provider
		capacityProvider, err := ecs.NewCapacityProvider(ctx, "pull-pulumi-capacity-provider", &ecs.CapacityProviderArgs{
			AutoScalingGroupProvider: ecs.CapacityProviderAutoScalingGroupProviderArgs{
				AutoScalingGroupArn: autoScalingGroup.Arn,
			},
		})
		if err != nil {
			return err
		}

		cluster, err := ecs.NewCluster(ctx, CLUSTER_NAME, &ecs.ClusterArgs{
			Name: pulumi.String(CLUSTER_NAME),
			CapacityProviders: pulumi.StringArray{
				capacityProvider.Name,
			},
		})
		if err != nil {
			return err
		}

		loadBalancer, err := awsxLb.NewApplicationLoadBalancer(ctx, "lb", &awsxLb.ApplicationLoadBalancerArgs{
			DefaultTargetGroupPort: pulumi.Int(3000),
		})
		if err != nil {
			return err
		}
		_, err = awsxEcs.NewEC2Service(ctx, SERVICE_NAME, &awsxEcs.EC2ServiceArgs{
			// Name: pulumi.String(SERVICE_NAME),
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
					Secrets: awsxEcs.TaskDefinitionSecretArray{
						awsxEcs.TaskDefinitionSecretArgs{
							Name:      pulumi.String("PRIVATE_KEY"),
							ValueFrom: pulumi.String(PRIVATE_KEY_ARN),
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
							Protocol:      pulumi.String("tcp"),
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
