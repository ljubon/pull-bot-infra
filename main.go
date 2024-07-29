package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
)

const (
	AWS_REGION			= "us-east-1"
	TAG_COST_VALUE	= "d-server-01068mdjl5jze3"
	BUCKET          = "arn:aws:s3:::pullbot-envs/.env" 

	CLOUD_WATCH_GROUP = "service-1"

	TASK_ROLE_ARN   = "arn:aws:iam::341894770476:role/ecsTaskExecutionRole"
	
	ECS_ROLE_ARN    = "arn:aws:iam::341894770476:instance-profile/ecsInstanceRole"
	
	CLUSTER_NAME    = "pull-pulumi-cluster"
	SERVICE_NAME    = "service"

	PULL_CONTAINER 		= "ghcr.io/ljubon/pull/pull:latest"
	// Private key generated from github app
	PRIVATE_KEY_ARN 	= "arn:aws:secretsmanager:us-east-1:341894770476:secret:LJUBOOPS_PRIVATE_KEY_ORIGINAL-LqUO7g"
	
	AMI_ID						= "ami-0e771da97cb597c23"
	EC2_PRIVATE_KEY 	= "pullbot"
	INSTANCE_TYPE			= "t2.medium"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		tags := pulumi.StringMap{
			// This is required for INFRA team to manage costs
			"map-migrated": pulumi.String(TAG_COST_VALUE),
		}

		// Create user data for EC2 instance, which will join the ECS cluster
		encodedUserData := pulumi.All("pull-pulumi-cluster").ApplyT(func(args []interface{}) (string, error) {
			userData := "#!bin/bash\necho ECS_CLUSTER=pull-pulumi-cluster >> /etc/ecs/ecs.config;"
			return base64.StdEncoding.EncodeToString([]byte(userData)), nil
		}).(pulumi.StringOutput)

		// Create Launch template for EC2 instance for our cluster
		launchTemplate, err := ec2.NewLaunchTemplate(ctx, "pull-pulumi-launch-template", &ec2.LaunchTemplateArgs{
			Name:         pulumi.String("pull-pulumi-launch-template"),
			ImageId:      pulumi.String(AMI_ID),
			InstanceType: pulumi.String(INSTANCE_TYPE),
			UserData:     encodedUserData,
			KeyName:      pulumi.String(EC2_PRIVATE_KEY),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileArgs{
				Arn: pulumi.String(ECS_ROLE_ARN),
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
		cluster, err := ecs.NewCluster(ctx, CLUSTER_NAME, &ecs.ClusterArgs{
			Name: pulumi.String(CLUSTER_NAME),
			Tags: tags,
		})
		if err != nil {
			return err
		}

		// Create a new CloudWatch Log Group
		cloudwatch.NewLogGroup(ctx, CLOUD_WATCH_GROUP, &cloudwatch.LogGroupArgs{
				Name: pulumi.String(CLOUD_WATCH_GROUP),
				RetentionInDays: pulumi.Int(7),
		})

		/*
			Creates a service which will deploy `DesiredCount` number of services with our container
			NOTE: Make sure to change `v1` to next version when changing something of this service
			This is needed so that we replace whole service
		**/
		serviceName := fmt.Sprintf("%s-v1", SERVICE_NAME)
		// Create Service & Task definition in ECS cluster
		awsxEcs.NewEC2Service(ctx, serviceName, &awsxEcs.EC2ServiceArgs{
			Name:         pulumi.String(SERVICE_NAME),
			Cluster:      cluster.Arn,
			DesiredCount: pulumi.Int(1),
			TaskDefinitionArgs: &awsxEcs.EC2ServiceTaskDefinitionArgs{
				NetworkMode: pulumi.String("host"),
				Container: &awsxEcs.TaskDefinitionContainerDefinitionArgs{
					Image:     pulumi.String(PULL_CONTAINER),
					Cpu:       pulumi.Int(1024),
					Memory:    pulumi.Int(2048),
					Essential: pulumi.Bool(true),
					LogConfiguration: &awsxEcs.TaskDefinitionLogConfigurationArgs{
						LogDriver: pulumi.String("awslogs"),
						Options: pulumi.StringMap{
							"awslogs-group":         pulumi.String(CLOUD_WATCH_GROUP),
							"awslogs-region":        pulumi.String(AWS_REGION),
							"awslogs-stream-prefix": pulumi.String("container"),
						},
					},
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
						},
					},
				},

				ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
					RoleArn: pulumi.String(TASK_ROLE_ARN),
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
