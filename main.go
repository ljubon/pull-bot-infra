package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	awsRegion     = os.Getenv("AWS_REGION")
	tagCostValue  = os.Getenv("TAG_COST_VALUE")
	bucket        = os.Getenv("BUCKET")
	taskRoleArn   = os.Getenv("TASK_ROLE_ARN")
	escRoleArn    = os.Getenv("ECS_ROLE_ARN")
	pullContainer = os.Getenv("PULL_CONTAINER")
	privateKeyArn = os.Getenv("PRIVATE_KEY_ARN")
	amiID         = os.Getenv("AMI_ID")
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		tags := pulumi.StringMap{
			// This is required for INFRA team to manage costs
			"map-migrated": pulumi.String(tagCostValue),
		}

		// Create user data for EC2 instance, which will join the ECS cluster
		encodedUserData := pulumi.All("pull-pulumi-cluster").ApplyT(func(args []interface{}) (string, error) {
			userData := "#!bin/bash\necho ECS_CLUSTER=pull-pulumi-cluster >> /etc/ecs/ecs.config;"
			return base64.StdEncoding.EncodeToString([]byte(userData)), nil
		}).(pulumi.StringOutput)

		// Create Launch template for EC2 instance for our cluster
		launchTemplate, err := ec2.NewLaunchTemplate(ctx, "pull-pulumi-launch-template", &ec2.LaunchTemplateArgs{
			Name:         pulumi.String("pull-pulumi-launch-template"),
			ImageId:      pulumi.String(amiID),
			InstanceType: pulumi.String("t2.medium"),
			UserData:     encodedUserData,
			KeyName:      pulumi.String("pullbot"),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileArgs{
				Arn: pulumi.String(escRoleArn),
			},
			Tags: tags,
		})
		if err != nil {
			return err
		}

		// Create EC2 instance with previously created launch template
		// Once EC2 is running, the agent will try join the ECS cluster and it will be shown under infrastructure tab
		ec2.NewInstance(ctx, "pull-pulumi-instance", &ec2.InstanceArgs{
			LaunchTemplate: ec2.InstanceLaunchTemplateArgs{
				Id:      launchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			Tags: tags,
		})

		// Create ECS cluster
		cluster, err := ecs.NewCluster(ctx, "pull-pulumi-cluster", &ecs.ClusterArgs{
			Name: pulumi.String("pull-pulumi-cluster"),
			Tags: tags,
		})
		if err != nil {
			return err
		}

		// Create a new CloudWatch Log Group
		cloudwatch.NewLogGroup(ctx, "service-1", &cloudwatch.LogGroupArgs{
			Name:            pulumi.String("service-1"),
			RetentionInDays: pulumi.Int(7),
		})

		/*
			Creates a service which will deploy `DesiredCount` number of services with our container
			NOTE: Make sure to change `v1` to next version when changing something of this service
			This is needed so that we replace whole service
		**/
		serviceName := fmt.Sprintf("%s-v1", "service")
		// Create Service & Task definition in ECS cluster
		awsxEcs.NewEC2Service(ctx, serviceName, &awsxEcs.EC2ServiceArgs{
			Name:         pulumi.String("service"),
			Cluster:      cluster.Arn,
			DesiredCount: pulumi.Int(1),
			TaskDefinitionArgs: &awsxEcs.EC2ServiceTaskDefinitionArgs{
				NetworkMode: pulumi.String("host"),
				Container: &awsxEcs.TaskDefinitionContainerDefinitionArgs{
					Image:     pulumi.String(pullContainer),
					Cpu:       pulumi.Int(1024),
					Memory:    pulumi.Int(2048),
					Essential: pulumi.Bool(true),
					LogConfiguration: &awsxEcs.TaskDefinitionLogConfigurationArgs{
						LogDriver: pulumi.String("awslogs"),
						Options: pulumi.StringMap{
							"awslogs-group":         pulumi.String("service-1"),
							"awslogs-region":        pulumi.String(awsRegion),
							"awslogs-stream-prefix": pulumi.String("container"),
						},
					},
					Secrets: awsxEcs.TaskDefinitionSecretArray{
						awsxEcs.TaskDefinitionSecretArgs{
							Name:      pulumi.String("PRIVATE_KEY"),
							ValueFrom: pulumi.String(privateKeyArn),
						},
					},
					EnvironmentFiles: awsxEcs.TaskDefinitionEnvironmentFileArray{
						awsxEcs.TaskDefinitionEnvironmentFileArgs{
							Type:  pulumi.String("s3"),
							Value: pulumi.String(bucket),
						},
					},
					PortMappings: awsxEcs.TaskDefinitionPortMappingArray{
						awsxEcs.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.Int(3000),
							HostPort:      pulumi.Int(3000),
							Protocol:      pulumi.String("tcp"),
						},
					},
				},

				ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
					RoleArn: pulumi.String(taskRoleArn),
				},
			},
			Tags: tags,
		},
			pulumi.DeleteBeforeReplace(true),
			pulumi.Aliases([]pulumi.Alias{
				{Type: pulumi.String("awsx:x:ecs:EC2Service"), Name: pulumi.String(serviceName)},
			}),
		)

		return nil
	})
}
