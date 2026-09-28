# AWS Deployment

This is the recommended AWS shape for the Counter Drop pilot:

- **ECR** for the Docker image built from the root `Dockerfile`.
- **ECS Fargate** for the app container.
- **RDS PostgreSQL 17** for the database.
- **S3** for uploaded print files.
- **Application Load Balancer + ACM** for HTTPS.
- **SSM Parameter Store or Secrets Manager** for `CD_DATABASE_URL`.
- **CloudWatch Logs** for app logs.

The Go API serves the built PWA from the same container, so the browser origin and API origin can be the same HTTPS URL.

## 1. Build Locally

```bash
make test
make build-image
```

The image name defaults to `counter-drop:local`.

## 2. Create AWS Resources

Create these once:

1. ECR repository: `counter-drop`.
2. S3 bucket for uploaded files, for example `counter-drop-prod-files`.
3. RDS PostgreSQL database.
4. ECS cluster and Fargate service behind an ALB.
5. ACM certificate for `app.your-domain.example`.
6. CloudWatch log group `/ecs/counter-drop-prod`.
7. ECS task execution role with the normal ECR/CloudWatch permissions plus `execution-role-secrets-policy.json` for SSM secrets.
8. ECS app task role with S3 permissions from `app-task-role-policy.json`.

For RDS, create a database/user and store the app URL in SSM:

```bash
aws ssm put-parameter \
  --name /counter-drop/prod/CD_DATABASE_URL \
  --type SecureString \
  --value 'postgres://counter_drop:<password>@<rds-endpoint>:5432/counter_drop?sslmode=require'
```

## 3. Configure S3 CORS

Edit `s3-cors.json` and replace the origin with your real HTTPS domain, then apply it:

```bash
aws s3api put-bucket-cors \
  --bucket counter-drop-prod-files \
  --cors-configuration file://deploy/aws/s3-cors.json
```

The app uses browser direct uploads to signed S3 URLs, so CORS must allow `PUT`, `GET`, and `HEAD` from the PWA origin.

## 4. Push Image To ECR

```bash
AWS_REGION=ap-south-1
AWS_ACCOUNT_ID=<account-id>
IMAGE_TAG=$(git rev-parse --short HEAD)

aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

docker build -t counter-drop:$IMAGE_TAG -f Dockerfile .
docker tag counter-drop:$IMAGE_TAG "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/counter-drop:$IMAGE_TAG"
docker push "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/counter-drop:$IMAGE_TAG"
```

## 5. Register ECS Task Definition

Copy `task-definition.json`, replace placeholders:

- `<account-id>`
- `<region>`
- `<image-tag>`
- `app.your-domain.example`
- `counter-drop-prod-files`
- IAM role names/ARNs

Then register it:

```bash
aws ecs register-task-definition \
  --cli-input-json file://deploy/aws/task-definition.json
```

Update the service:

```bash
aws ecs update-service \
  --cluster counter-drop-prod \
  --service counter-drop-prod \
  --task-definition counter-drop-prod
```

## 6. Required Runtime Env

Set these in the task definition:

```text
CD_ENV=prod
CD_PUBLIC_API_URL=https://app.your-domain.example
CD_PUBLIC_WEB_URL=https://app.your-domain.example
CD_WEB_ORIGINS=https://app.your-domain.example
CD_STORAGE_REGION=ap-south-1
CD_STORAGE_BUCKET=counter-drop-prod-files
CD_DEMO_SEED=false
```

Set this as a secret from SSM/Secrets Manager:

```text
CD_DATABASE_URL=postgres://...
```

Do not set `CD_STORAGE_ACCESS_KEY` or `CD_STORAGE_SECRET_KEY` on ECS when using the app task role. The AWS SDK will use the task role automatically.

## 7. Health Checks

The ALB target group should call:

```text
GET /ready
```

Use `/health` for simple uptime checks and `/ready` for database readiness.

## 8. First Shop

Run `cdadmin` from your laptop or a one-off ECS task with network access to RDS:

```bash
CD_DATABASE_URL='postgres://counter_drop:<password>@<rds-endpoint>:5432/counter_drop?sslmode=require' \
CD_PUBLIC_WEB_URL='https://app.your-domain.example' \
./bin/cdadmin create-shop \
  -slug first-shop \
  -name 'First Shop' \
  -address 'Shop Address' \
  -owner 'Owner Name'
```

Send the printed setup link to the owner.

## 9. Go/No-Go AWS Checklist

- `https://app.your-domain.example/health` returns 200.
- `https://app.your-domain.example/ready` returns 200.
- ECS task logs show migrations applied and no storage errors.
- Upload URLs point to S3, not localhost.
- S3 CORS allows phone uploads from the PWA domain.
- RDS has automated backups enabled.
- S3 bucket has lifecycle/retention policy appropriate for your deletion promise.
- Phone QR test works on Android Chrome and iPhone Safari.
