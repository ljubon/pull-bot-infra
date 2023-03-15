import * as aws from "@pulumi/aws";
import * as awsx from "@pulumi/awsx";


export interface ECSResources {
    cluster: aws.ecs.Cluster
    service: awsx.ecs.EC2Service
    loadBalancer: awsx.lb.ApplicationLoadBalancer
}

export function createECSResources(): ECSResources {
    const vpc = new awsx.ec2.Vpc("pullbot-vpc", {})
    const securityGroup = new aws.ec2.SecurityGroup(
        "pullbot-security-group", {
            vpcId: vpc.vpcId,
            egress: [{
                fromPort: 0,
                toPort: 0,
                protocol: "-1",
                cidrBlocks: ["0.0.0.0/0"],
                ipv6CidrBlocks: ["::/0"]
            }]
        })
    
    const cluster = new aws.ecs.Cluster("pullbot-cluster", {})
    const loadBalancer = new awsx.lb.ApplicationLoadBalancer("pullbot-lb", {})
    const service = new awsx.ecs.EC2Service(
        "pullbot-service", {
            cluster: cluster.arn,
            desiredCount: 5,
            networkConfiguration: {
                subnets: vpc.privateSubnetIds,
                securityGroups: [securityGroup.id]
            },
            taskDefinitionArgs: {
                container: {
                    name: "pullbot:latest",
                    image: "ghcr.io/ljubon/pull/pull:latest",
                    cpu: 512,
                    memory: 128,
                    essential: true,
                    portMappings: [{
                        containerPort: 3000,
                        hostPort: 3000,
                        targetGroup: loadBalancer.defaultTargetGroup
                    }]
                }
            }  
        })
    return {
        cluster,
        service,
        loadBalancer
    }
}
