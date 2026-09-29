# AWS Deployment

> **Interim path.** The target is Lambda + DynamoDB built by a CDK stack (ADR-001, `docs/` in the project). Until that stack exists, this page runs the same container on ECS Fargate against the same DynamoDB table, so data carries over unchanged when you move to Lambda.

This AWS shape runs the Counter Drop container:

- **ECR** for the Docker image built from the root `Dockerfile`.
- **ECS Fargate** for the app container.
- **DynamoDB** table `cd-main` for all data (no database server, no passwords: access is through the task role).
- **S3** for uploaded print files.
- **Application Load Balancer + ACM** for HTTPS.
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
3. DynamoDB table `cd-main` (below).
4. ECS cluster and Fargate service behind an ALB.
5. ACM certificate for `app.your-domain.example`.
6. CloudWatch log group `/ecs/counter-drop-prod`.
7. ECS task execution role with the normal ECR/CloudWatch permissions (no secrets to read).
8. ECS app task role with DynamoDB and S3 permissions from `app-task-role-policy.json` (fill in region and account).

Create the table once, with your own AWS credentials (it creates the four indexes, turns on TTL and 7-day point-in-time recovery):

```bash
cd api && CD_DYNAMODB_TABLE=cd-main CD_DYNAMODB_REGION=ap-south-1 go run ./cmd/cdadmin create-table
```

## 3. Configure S3 CORS

Edit `s3-cors.json` and replace the origin with your real HTTPS domain, then apply it:

```bash
aws s3api put-bucket-cors \
  --bucket counter-drop-prod-files \
  --cors-configuration file://deploy/aws/s3-cors.json
```

The app uses browser direct uploads to signed S3 URLs, so CORS must allow `PUT`, `GET`, and `HEAD` from the PWA origin.

### Bucket safety (do this once)

The deletion promise depends on these:

```bash
# No public access, ever.
aws s3api put-public-access-block --bucket counter-drop-prod-files \
  --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

# Backstop: anything left under cd/ is removed after 8 days even if the app's deletion worker stopped.
aws s3api put-bucket-lifecycle-configuration --bucket counter-drop-prod-files \
  --lifecycle-configuration file://deploy/aws/s3-lifecycle.json
```

**Leave versioning off** (the default) and don't add replication: with versioning on, a "deleted" file survives as an old version.

## 4. Push Image To ECR

```bash
AWS_REGION=ap-south-1
AWS_ACCOUNT_ID=<account-id>
IMAGE_TAG=$(git rev-parse --short HEAD)

aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

docker build -t counter-drop:$IMAGE_TAG -f Dockerfile \
  --build-arg VITE_OPERATOR_NAME="Your Company Name" \
  --build-arg VITE_SUPPORT_EMAIL="support@your-domain.example" .
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

### Run exactly one task

Set the ECS service's **desired count to 1**. Live updates and rate limits are kept in the container's memory, so a second task would miss events and double the limits. One Fargate task (0.5 vCPU / 1 GB) handles a pilot comfortably. Two tasks overlapping during a deploy is safe for data: every change is a conditional write on DynamoDB.

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
CD_TRUST_PROXY=true   # the ALB sets X-Forwarded-For; rate limits need the real client IP
CD_RATE_LIMIT=true
```

The privacy/terms contact is baked in at image build time (`VITE_OPERATOR_NAME`, `VITE_SUPPORT_EMAIL`; for the GitHub workflow set them as repository variables).

```text
CD_DYNAMODB_TABLE=cd-main
CD_DYNAMODB_REGION=ap-south-1
```

There are no database secrets. Do not set `CD_STORAGE_ACCESS_KEY` or `CD_STORAGE_SECRET_KEY` on ECS when using the app task role. The AWS SDK uses the task role automatically, for S3 and DynamoDB.

## 7. Health Checks

The ALB target group should call:

```text
GET /ready
```

Use `/health` for simple uptime checks and `/ready` for readiness (it checks the DynamoDB table).

## 8. First Shop

Run `cdadmin` from your laptop with your AWS credentials (no network access to a database server is needed):

```bash
CD_DYNAMODB_TABLE=cd-main CD_DYNAMODB_REGION=ap-south-1 \
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
- ECS task logs show "store ready" and no storage errors.
- Upload URLs point to S3, not localhost.
- S3 CORS allows phone uploads from the PWA domain.
- DynamoDB table has point-in-time recovery on (7 days) and TTL on `ttl` (`cdadmin create-table` does both).
- S3 bucket: public access blocked, versioning off, `s3-lifecycle.json` applied.
- ECS service desired count is 1.
- `/privacy` shows your company name and support email.
- Phone QR test works on Android Chrome and iPhone Safari.
