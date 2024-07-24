package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	PULL_CONTAINER  = "ghcr.io/ljubon/pull/pull:latest"
	BUCKET          = "arn:aws:s3:::pullbot-envs/.env"
	PRIVATE_KEY_ARN = "arn:aws:secretsmanager:us-east-1:341894770476:secret:LJUBOOPS_PRIVATE_KEY_ORIGINAL-LqUO7g" // ljuboops private key for github app
	TASK_ROLE_ARN   = "arn:aws:iam::341894770476:role/ecsTaskExecutionRole"
	TASK_ROLE_NAME  = "ecsTaskExecutionRole"
	ECS_ROLE_ARN    = "arn:aws:iam::341894770476:instance-profile/ecsInstanceRole"
	ECS_ROLE_NAME   = "ecsInstanceRole"
	VPC_ID          = "vpc-0fbca88fc6fab7a0f"
	SECURITY_GROUP  = "sg-01a8e31f04b83e53d"
	CLUSTER_NAME    = "pull-pulumi-cluster"
	SERVICE_NAME    = "service"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		tags := pulumi.StringMap{
			"map-migrated": pulumi.String("d-server-01068mdjl5jze3"),
		}

		encodedUserData := pulumi.All("pull-pulumi-cluster").ApplyT(func(args []interface{}) (string, error) {
			userData := "#!bin/bash\necho ECS_CLUSTER=pull-pulumi-cluster >> /etc/ecs/ecs.config;"
			return base64.StdEncoding.EncodeToString([]byte(userData)), nil
		}).(pulumi.StringOutput)

		launchTemplate, err := ec2.NewLaunchTemplate(ctx, "pull-pulumi-launch-template", &ec2.LaunchTemplateArgs{
			Name:         pulumi.String("pull-pulumi-launch-template"),
			// ImageId:      pulumi.String("ami-0c76be34ffbfb0b14"), 		// amzn2-ami-ecs-hvm-2.0.20230321-x86_64-ebs
			ImageId:      pulumi.String("ami-0e771da97cb597c23"), 	// amzn2-ami-ecs-hvm-2.0.20240227-x86_64-ebs
			InstanceType: pulumi.String("t2.medium"),
			UserData:     encodedUserData,
			KeyName:      pulumi.String("pullbot"),
			VpcSecurityGroupIds: pulumi.StringArray{
				pulumi.String(SECURITY_GROUP),
			},
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileArgs{
				Arn: pulumi.String(ECS_ROLE_ARN),
			},
			Tags: tags,
		})
		if err != nil {
			return err
		}

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

		// NOTE: Make sure to change `v1` to next version when changing something of this service
		// This is needed so that we replace whole service
		serviceName := fmt.Sprintf("%s-v1", SERVICE_NAME)
		// Create Service & Task definition in ECS cluster
		awsxEcs.NewEC2Service(ctx, serviceName, &awsxEcs.EC2ServiceArgs{
			Name:         pulumi.String(SERVICE_NAME),
			Cluster:      cluster.Arn,
			DesiredCount: pulumi.Int(1),
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
					Cpu:       pulumi.Int(1024),
					Memory:    pulumi.Int(2048),
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
						},
					},
				},
				// NOTE: When deploying fresh infra, deploy with commenting this line, after deploy enable this mode
				NetworkMode: pulumi.String("host"),

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
