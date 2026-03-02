# AWS Deployment Runbook (End-to-End)

This guide deploys on AWS using ECS Fargate:

- `backend` ECS service behind ALB
- `frontend` ECS service behind ALB
- EFS for persistent SQLite file (no GCP dependency)
- ECR for container images
- Secrets Manager + SSM Parameter Store for configuration

> This path is intentionally SQLite-based because this codebase currently supports `firestore` and `sqlite` stores.

## 1) Prerequisites

- AWS CLI v2 configured (`aws configure`)
- `docker` installed and logged in
- Existing VPC + 2+ subnets in same region

Set shell variables:

```bash
export AWS_REGION="ap-south-1"
export ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
export CLUSTER="sync-audio-cluster"
export BACKEND_REPO="sync-audio-backend"
export FRONTEND_REPO="sync-audio-frontend"
export BACKEND_SERVICE="sync-audio-backend"
export FRONTEND_SERVICE="sync-audio-frontend"
export VPC_ID="vpc-xxxxxxxx"
export SUBNET_A="subnet-aaaaaaaa"
export SUBNET_B="subnet-bbbbbbbb"
```

## 2) Create ECR repositories

```bash
aws ecr create-repository --repository-name "$BACKEND_REPO" --region "$AWS_REGION"
aws ecr create-repository --repository-name "$FRONTEND_REPO" --region "$AWS_REGION"
```

Set image URLs:

```bash
export BACKEND_IMAGE="$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$BACKEND_REPO:latest"
export FRONTEND_IMAGE="$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$FRONTEND_REPO:latest"
```

Login and push:

```bash
aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

docker build -t "$BACKEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/backend
docker push "$BACKEND_IMAGE"

docker build -t "$FRONTEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/frontend
docker push "$FRONTEND_IMAGE"
```

## 3) Create ECS cluster

```bash
aws ecs create-cluster --cluster-name "$CLUSTER" --region "$AWS_REGION"
```

## 4) Create security groups

```bash
export ALB_SG="$(aws ec2 create-security-group --group-name sync-audio-alb-sg --description 'ALB SG' --vpc-id "$VPC_ID" --query GroupId --output text --region "$AWS_REGION")"
export ECS_SG="$(aws ec2 create-security-group --group-name sync-audio-ecs-sg --description 'ECS SG' --vpc-id "$VPC_ID" --query GroupId --output text --region "$AWS_REGION")"
export EFS_SG="$(aws ec2 create-security-group --group-name sync-audio-efs-sg --description 'EFS SG' --vpc-id "$VPC_ID" --query GroupId --output text --region "$AWS_REGION")"
```

Rules:

```bash
aws ec2 authorize-security-group-ingress --group-id "$ALB_SG" --protocol tcp --port 80 --cidr 0.0.0.0/0 --region "$AWS_REGION"
aws ec2 authorize-security-group-ingress --group-id "$ECS_SG" --protocol tcp --port 8080 --source-group "$ALB_SG" --region "$AWS_REGION"
aws ec2 authorize-security-group-ingress --group-id "$ECS_SG" --protocol tcp --port 3000 --source-group "$ALB_SG" --region "$AWS_REGION"
aws ec2 authorize-security-group-ingress --group-id "$EFS_SG" --protocol tcp --port 2049 --source-group "$ECS_SG" --region "$AWS_REGION"
```

## 5) Create EFS for SQLite persistence

```bash
export EFS_ID="$(aws efs create-file-system --creation-token sync-audio-efs --performance-mode generalPurpose --query FileSystemId --output text --region "$AWS_REGION")"
aws efs create-mount-target --file-system-id "$EFS_ID" --subnet-id "$SUBNET_A" --security-groups "$EFS_SG" --region "$AWS_REGION"
aws efs create-mount-target --file-system-id "$EFS_ID" --subnet-id "$SUBNET_B" --security-groups "$EFS_SG" --region "$AWS_REGION"
```

## 6) Store secrets/config in AWS

Generate locally:

```bash
openssl rand -hex 32
openssl rand -base64 32
```

Create secrets in Secrets Manager:

```bash
aws secretsmanager create-secret --name sync-audio/SESSION_HMAC_KEY --secret-string "<SESSION_HMAC_KEY_VALUE>" --region "$AWS_REGION"
aws secretsmanager create-secret --name sync-audio/TOKEN_ENCRYPTION_KEY --secret-string "<TOKEN_ENCRYPTION_KEY_VALUE>" --region "$AWS_REGION"
aws secretsmanager create-secret --name sync-audio/TURNSTILE_SECRET_KEY --secret-string "<TURNSTILE_SECRET_OR_EMPTY>" --region "$AWS_REGION"
```

Create SSM parameters:

```bash
aws ssm put-parameter --name /sync-audio/STORE_PROVIDER --type String --value sqlite --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/SQLITE_PATH --type String --value /data/sync-audio.db --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/DEFAULT_RATE_LIMIT_PER_MIN --type String --value 120 --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/SIGNUP_RATE_LIMIT_PER_HOUR --type String --value 10 --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/ACCESS_CODE_MAX_USES --type String --value 1 --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/ACCESS_CODE_MAX_FAILURES --type String --value 5 --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/ACCESS_CODE_LOCKOUT_MINUTES --type String --value 15 --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/ADMIN_EMAILS --type String --value "dev@example.com" --overwrite --region "$AWS_REGION"
aws ssm put-parameter --name /sync-audio/SIGNUP_ACCESS_CODES --type String --value "Ryanthisisforoouuuuu" --overwrite --region "$AWS_REGION"
```

## 7) Create IAM roles for ECS tasks

Create trust policy file `/tmp/ecs-trust.json`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ecs-tasks.amazonaws.com" },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

Create roles:

```bash
aws iam create-role --role-name sync-audio-ecs-task-exec-role --assume-role-policy-document file:///tmp/ecs-trust.json
aws iam attach-role-policy --role-name sync-audio-ecs-task-exec-role --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

aws iam create-role --role-name sync-audio-ecs-task-role --assume-role-policy-document file:///tmp/ecs-trust.json
```

Attach inline policy (Secrets + SSM + EFS client):

```bash
cat >/tmp/sync-audio-task-policy.json <<'JSON'
{
  "Version":"2012-10-17",
  "Statement":[
    {"Effect":"Allow","Action":["secretsmanager:GetSecretValue"],"Resource":"*"},
    {"Effect":"Allow","Action":["ssm:GetParameter","ssm:GetParameters"],"Resource":"*"},
    {"Effect":"Allow","Action":["elasticfilesystem:ClientMount","elasticfilesystem:ClientWrite","elasticfilesystem:ClientRootAccess"],"Resource":"*"}
  ]
}
JSON
aws iam put-role-policy --role-name sync-audio-ecs-task-role --policy-name sync-audio-task-inline --policy-document file:///tmp/sync-audio-task-policy.json
```

## 8) Create ALB + target groups + listeners

```bash
export ALB_ARN="$(aws elbv2 create-load-balancer --name sync-audio-alb --subnets "$SUBNET_A" "$SUBNET_B" --security-groups "$ALB_SG" --query LoadBalancers[0].LoadBalancerArn --output text --region "$AWS_REGION")"
export ALB_DNS="$(aws elbv2 describe-load-balancers --load-balancer-arns "$ALB_ARN" --query LoadBalancers[0].DNSName --output text --region "$AWS_REGION")"

export TG_BACKEND="$(aws elbv2 create-target-group --name sync-audio-backend-tg --protocol HTTP --port 8080 --target-type ip --vpc-id "$VPC_ID" --health-check-path /v1/health --query TargetGroups[0].TargetGroupArn --output text --region "$AWS_REGION")"
export TG_FRONTEND="$(aws elbv2 create-target-group --name sync-audio-frontend-tg --protocol HTTP --port 3000 --target-type ip --vpc-id "$VPC_ID" --health-check-path / --query TargetGroups[0].TargetGroupArn --output text --region "$AWS_REGION")"
```

Listener + path routing:

```bash
export LISTENER_ARN="$(aws elbv2 create-listener --load-balancer-arn "$ALB_ARN" --protocol HTTP --port 80 --default-actions Type=forward,TargetGroupArn="$TG_FRONTEND" --query Listeners[0].ListenerArn --output text --region "$AWS_REGION")"
aws elbv2 create-rule --listener-arn "$LISTENER_ARN" --priority 10 --conditions Field=path-pattern,Values='/v1/*' --actions Type=forward,TargetGroupArn="$TG_BACKEND" --region "$AWS_REGION"
```

## 9) Register ECS task definitions

Create backend task file `/tmp/backend-task.json`:

```json
{
  "family": "sync-audio-backend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/sync-audio-ecs-task-exec-role",
  "taskRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/sync-audio-ecs-task-role",
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "<BACKEND_IMAGE>",
      "essential": true,
      "portMappings": [{ "containerPort": 8080, "protocol": "tcp" }],
      "environment": [
        { "name": "APP_ENV", "value": "production" },
        { "name": "PORT", "value": "8080" },
        { "name": "ALLOWED_ORIGIN", "value": "http://<ALB_DNS>" },
        { "name": "STORE_PROVIDER", "value": "sqlite" },
        { "name": "SQLITE_PATH", "value": "/data/sync-audio.db" },
        { "name": "FIRESTORE_PROJECT_ID", "value": "unused-for-sqlite" },
        { "name": "DEFAULT_RATE_LIMIT_PER_MIN", "value": "120" },
        { "name": "SIGNUP_RATE_LIMIT_PER_HOUR", "value": "10" },
        { "name": "ACCESS_CODE_MAX_USES", "value": "1" },
        { "name": "ACCESS_CODE_MAX_FAILURES", "value": "5" },
        { "name": "ACCESS_CODE_LOCKOUT_MINUTES", "value": "15" },
        { "name": "ADMIN_EMAILS", "value": "dev@example.com" },
        { "name": "SIGNUP_ACCESS_CODES", "value": "Ryanthisisforoouuuuu" }
      ],
      "secrets": [
        { "name": "SESSION_HMAC_KEY", "valueFrom": "arn:aws:secretsmanager:<REGION>:<ACCOUNT_ID>:secret:sync-audio/SESSION_HMAC_KEY" },
        { "name": "TOKEN_ENCRYPTION_KEY", "valueFrom": "arn:aws:secretsmanager:<REGION>:<ACCOUNT_ID>:secret:sync-audio/TOKEN_ENCRYPTION_KEY" },
        { "name": "TURNSTILE_SECRET_KEY", "valueFrom": "arn:aws:secretsmanager:<REGION>:<ACCOUNT_ID>:secret:sync-audio/TURNSTILE_SECRET_KEY" }
      ],
      "mountPoints": [
        { "sourceVolume": "sqlite-data", "containerPath": "/data", "readOnly": false }
      ]
    }
  ],
  "volumes": [
    {
      "name": "sqlite-data",
      "efsVolumeConfiguration": {
        "fileSystemId": "<EFS_ID>",
        "rootDirectory": "/",
        "transitEncryption": "ENABLED"
      }
    }
  ]
}
```

Replace placeholders then register:

```bash
sed -i "s|<ACCOUNT_ID>|$ACCOUNT_ID|g; s|<REGION>|$AWS_REGION|g; s|<BACKEND_IMAGE>|$BACKEND_IMAGE|g; s|<EFS_ID>|$EFS_ID|g; s|<ALB_DNS>|$ALB_DNS|g" /tmp/backend-task.json
aws ecs register-task-definition --cli-input-json file:///tmp/backend-task.json --region "$AWS_REGION"
```

Create frontend task file `/tmp/frontend-task.json`:

```json
{
  "family": "sync-audio-frontend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/sync-audio-ecs-task-exec-role",
  "taskRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/sync-audio-ecs-task-role",
  "containerDefinitions": [
    {
      "name": "frontend",
      "image": "<FRONTEND_IMAGE>",
      "essential": true,
      "portMappings": [{ "containerPort": 3000, "protocol": "tcp" }],
      "environment": [
        { "name": "NEXT_PUBLIC_API_BASE_URL", "value": "http://<ALB_DNS>" },
        { "name": "NEXT_PUBLIC_TURNSTILE_SITE_KEY", "value": "<TURNSTILE_SITE_KEY_OR_EMPTY>" }
      ]
    }
  ]
}
```

Replace placeholders then register:

```bash
sed -i "s|<ACCOUNT_ID>|$ACCOUNT_ID|g; s|<FRONTEND_IMAGE>|$FRONTEND_IMAGE|g; s|<ALB_DNS>|$ALB_DNS|g" /tmp/frontend-task.json
aws ecs register-task-definition --cli-input-json file:///tmp/frontend-task.json --region "$AWS_REGION"
```

## 10) Create ECS services

```bash
aws ecs create-service \
  --cluster "$CLUSTER" \
  --service-name "$BACKEND_SERVICE" \
  --task-definition sync-audio-backend \
  --desired-count 1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_A,$SUBNET_B],securityGroups=[$ECS_SG],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=$TG_BACKEND,containerName=backend,containerPort=8080" \
  --region "$AWS_REGION"

aws ecs create-service \
  --cluster "$CLUSTER" \
  --service-name "$FRONTEND_SERVICE" \
  --task-definition sync-audio-frontend \
  --desired-count 1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_A,$SUBNET_B],securityGroups=[$ECS_SG],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=$TG_FRONTEND,containerName=frontend,containerPort=3000" \
  --region "$AWS_REGION"
```

## 11) Verify deployment

Use ALB DNS:

```bash
echo "http://$ALB_DNS"
curl "http://$ALB_DNS/v1/health"
curl "http://$ALB_DNS/v1/docs/openapi.yaml" | head -n 5
curl -I "http://$ALB_DNS/"
```

Manual check:

- open `http://<ALB_DNS>/`
- run login/request-access flow
- visit `http://<ALB_DNS>/v1/docs`

## 12) Updates and rollbacks

Update images and force new deployments:

```bash
docker build -t "$BACKEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/backend
docker push "$BACKEND_IMAGE"
aws ecs update-service --cluster "$CLUSTER" --service "$BACKEND_SERVICE" --force-new-deployment --region "$AWS_REGION"

docker build -t "$FRONTEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/frontend
docker push "$FRONTEND_IMAGE"
aws ecs update-service --cluster "$CLUSTER" --service "$FRONTEND_SERVICE" --force-new-deployment --region "$AWS_REGION"
```

## 13) AWS hardening checklist

- Put ALB behind CloudFront + WAF for public traffic.
- Use ACM + HTTPS listener on ALB.
- Restrict admin emails and rotate secrets in Secrets Manager.
- Add CloudWatch alarms for ECS task restarts, ALB 5xx, latency.
- Use private subnets + NAT + tighter SG rules for production.
