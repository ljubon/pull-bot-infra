# pull-bot-infra

IaC implementation for self-hosting pull bot on AWS.

## AWS information

The infrastructure will be deployed to the `gross-devops-$env` account on AWS.

Login to Account ID `isc-login` and your personal account, then switch role to `gross-devops-$env`.

- DEV: gross-devops-dev: <https://signin.aws.amazon.com/switchrole?account=gross-devops-dev&roleName=isc-login_assumed-role_eng_admins&displayName=gross-devops-dev%20eng_admins>
- PRD: gross-devops-prd <https://signin.aws.amazon.com/switchrole?account=gross-devops-prd&roleName=isc-login_assumed-role_eng_admins&displayName=gross-devops-prod%20eng_admins>

### Prerequisites for PullBot Infrastructure

__NOTE__ This should be manually created before running IaC code.

#### Create IAM user `pull-bot`

- IAM -> Users -> Add user
- No console access
- Name `pull-bot`
- Create access keys (third-party)
- create `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
- attach policy + copy JSON direct inline policy (use from `gross-devops-dev` or `gross-devops-prd` account)

#### Create an S3 bucket for backend/state pulumi
  
- S3 -> Create bucket
- Name: `pull-bot-infra-backend-$env`
- Public access allowed
- To verify bucket exists

  ```bash
  aws s3 ls pull-bot-infra-backend-$env
  ```

#### Create bucket pullbot-envs-$env

- Create `.env` file and upload to the bucket (example content below)

  ```bash
    APP_ID=120461                   // GitHub App ID used for pull-bot
    APP_NAME=ljuboops               // GitHub App name
    WEBHOOK_SECRET=ljuboops         // GitHub App webhook secret
    MERGE_UNSTABLE=true             // Merge unstable PRs [default: false]
    PULL_INTERVAL=10                // Pull interval in seconds [default: 3600]
    JOB_TIMEOUT=30                  // Job timeout in seconds [default: 60]
    DEFAULT_MERGE_METHOD=hardreset  // Default merge method [default: hardreset - merge, squash, rebase]
  ```

#### Pulumi Login

Create a secret in AWS secret manager for private key

- Go to your App Settings -> Private keys -> Generate a private key
  - <https://github.com/settings/apps/ljuboops>
  - Download the private key
- Name: `$githubAppName-app-private-key`
  - Paste content of private key

#### Create ECS role (`ecsTaskExecutionRole`)

- add `ecs-to-ec2` inline policy + AmazonEc2ContainerServiceEC2Role
  - Copy from `gross-devops-dev` or `gross-devops-prd`

#### Create key pair

- EC2 -> key pair
- set name to `pullbot`

#### Create env/secrets on GitHub

- Go to <https://github.com/ljubon/pull-bot-infra/settings/environments>
- Create new environment
- Add secrets:
  - AWS_ACCESS_KEY_ID
  - AWS_SECRET_ACCESS_KEY
  - AWS_REGION
  - PULUMI_BACKEND_URL
  - PULUMI_CONFIG_PASSPHRASE

#### Default infrastructure information, no action required

- VPC_ID (default)
- SUBNET_ID (default)
- SECURITY_GROUP (default)

#### Run Pulumi workflow

- Go to <https://github.com/ljubon/pull-bot-infra/actions/workflows/pulumi.yml>
- Run workflow
  - Branch: `gross-devops-$env`
  - Select Pulumi actions: `preview | up | destroy`
  - Pulumi refresh: `true | false`
  - Select pulumi stack: `dev` (for now all stacks are created as `dev` stack)
