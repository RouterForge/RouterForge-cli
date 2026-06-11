#!/bin/bash
# Deploy big-pickle to ECS (Fargate)
set -euo pipefail

# --- Configuration ---
AWS_REGION="us-east-1"
ECR_REPOSITORY="big-pickle"
ECS_CLUSTER="big-pickle-cluster"
ECS_SERVICE="big-pickle-service"
TASK_DEF_FAMILY="big-pickle"
IMAGE_TAG="${1:-latest}"

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}"

# --- Build & Push ---
echo "Building Docker image..."
docker build -t "${ECR_REPOSITORY}:${IMAGE_TAG}" .

docker tag "${ECR_REPOSITORY}:${IMAGE_TAG}" "${ECR_URI}:${IMAGE_TAG}"

aws ecr get-login-password --region "${AWS_REGION}" | \
  docker login --username AWS --password-stdin "${ECR_URI}"

echo "Pushing image to ECR..."
docker push "${ECR_URI}:${IMAGE_TAG}"

# --- Update ECS Task Definition ---
TASK_DEF_JSON=$(aws ecs describe-task-definition \
  --task-definition "${TASK_DEF_FAMILY}" \
  --query 'taskDefinition' \
  --region "${AWS_REGION}")

NEW_TASK_DEF=$(echo "${TASK_DEF_JSON}" | \
  jq '.containerDefinitions[0].image="'${ECR_URI}:${IMAGE_TAG}'"' | \
  jq 'del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)')

aws ecs register-task-definition \
  --cli-input-json "$(echo "${NEW_TASK_DEF}")" \
  --region "${AWS_REGION}" > /dev/null

# --- Trigger Service Update ---
aws ecs update-service \
  --cluster "${ECS_CLUSTER}" \
  --service "${ECS_SERVICE}" \
  --task-definition "${TASK_DEF_FAMILY}" \
  --force-new-deployment \
  --region "${AWS_REGION}" > /dev/null

echo "Deployment to ECS triggered successfully."