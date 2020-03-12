#!/usr/bin/env bash

set -ueo pipefail

AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-"us-west-2"}

#######################################
# Create ECR repository if not exists
# Globals:
#   AWS_DEFAULT_REGION
# Arguments:
#   img_repo
#
# sample: ecr::ensure_repository amazon/app-mesh-controller
ecr::ensure_repository() {
    declare -r img_repo="$1"
    if ! aws ecr describe-repositories \
            --region=${AWS_DEFAULT_REGION} \
            --repository-names "${img_repo}" >/dev/null 2>&1; then
        echo "creating ECR repo with name ${img_repo}"
        aws ecr create-repository --region ${AWS_DEFAULT_REGION} --repository-name "${img_repo}"
    fi
}


#######################################
# Generate docker image name for ecr
# Globals:
#   AWS_DEFAULT_REGION
# Arguments:
#   img_repo
#   img_tag
#
# sample: ecr::name_image aws-alb-ingress-controlle v1.0.0 image_name
#######################################
ecr::name_image() {
    declare -r img_repo="$1" img_tag="$2"

    local aws_account_id=$(aws sts get-caller-identity --region ${AWS_DEFAULT_REGION} --query Account --output text)
    if [[ -z "$aws_account_id" ]]; then
        echo "Unable to get AWS account ID" >&2
        return 1
    fi

    echo "$aws_account_id.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com/$img_repo:$img_tag"
}

#######################################
# Check whether docker image exists in ECR
# Globals:
#   AWS_DEFAULT_REGION
# Arguments:
#   img_repo
#   img_tag
#
# sample: ecr::contains_image image_name
#######################################
ecr::contains_image() {
  declare -r img_repo="$1" img_tag="$2"

  aws ecr describe-images --region=${AWS_DEFAULT_REGION} --repository-name "${img_repo}" --image-ids imageTag="${img_tag}" 2>/dev/null
}

#######################################
# Push docker image to ECR
# Globals:
#   AWS_DEFAULT_REGION
# Arguments:
#   img_name
#
# sample: ecr::push_image image_name
#######################################
ecr::push_image() {
    declare -r img_name="$1"

    eval $(aws ecr get-login --region ${AWS_DEFAULT_REGION} --no-include-email)
    docker push "$img_name"
}

