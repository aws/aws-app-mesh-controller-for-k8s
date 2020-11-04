#!/usr/bin/env bash

AWS_CLI_VERSION=$(aws --version 2>&1 | cut -d/ -f2 | cut -d. -f1)

# aws_check_credentials() calls the STS::GetCallerIdentity API call and
# verifies that there is a local identity for running AWS commands
aws_check_credentials() {
    echo -n "checking AWS credentials ... "
    aws sts get-caller-identity --query "Account" >/dev/null ||
        ( echo "\nFATAL: No AWS credentials found. Please run \`aws configure\` to set up the CLI for your credentials." && exit 1 )
    echo "ok."
}

aws_account_id() {
    JSON=$( aws sts get-caller-identity --output json || exit 1 )
    echo "${JSON}" | jq --raw-output ".Account"
}

ecr_login() {
    echo -n "ecr login ... "
    local __aws_region=$1
    local __ecr_url=$2
    local __err_msg="Failed ECR login. Please make sure you have IAM permissions to access ECR."
    if [ $AWS_CLI_VERSION -gt 1 ]; then
        ( aws ecr get-login-password --region $__aws_region | \
                docker login --username AWS --password-stdin $__ecr_url ) ||
                ( echo "\n$__err_msg" && exit 1 )
    else
        $( aws ecr get-login --no-include-email ) || ( echo "\n$__err_msg" && exit 1 )
    fi
    echo "ok."
}
