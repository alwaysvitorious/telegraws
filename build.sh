#!/bin/bash

# Call from root directory: ./build.sh [--local|--lambda]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$SCRIPT_DIR/bin"
CONFIG_FILE="$SCRIPT_DIR/config/config.json"

# Parse command line arguments
MODE="lambda" # default
if [ "$1" = "--local" ]; then
    MODE="local"
elif [ "$1" = "--lambda" ]; then
    MODE="lambda"
fi

get_function_name() {
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "❌ Config file not found: $CONFIG_FILE"
        echo "💡 Make sure you have a config.json file in the root directory"
        exit 1
    fi

    FUNCTION_NAME=$(grep '"lambdaFunctionName"' "$CONFIG_FILE" |
        cut -d':' -f2 |
        tr -d ' ",' |
        head -n1)

    if [ "$FUNCTION_NAME" = "" ]; then
        echo "❌ Function name not found in config.json"
        echo "💡 Make sure global.deployment.lambdaFunctionName is set in your config.json"
        echo "💡 Expected format: \"lambdaFunctionName\": \"your-function-name\""
        exit 1
    fi

    echo "📋 Using function name from config: $FUNCTION_NAME"
}

get_cron_expression() {
    CRON_EXPRESSION=$(grep '"lambdaCronExpression"' "$CONFIG_FILE" |
        cut -d':' -f2 |
        tr -d '",' |              # Remove quotes and commas, keep spaces
        sed 's/^[[:space:]]*//' | # Trim leading spaces
        head -n1)

    if [ "$CRON_EXPRESSION" = "" ]; then
        echo "❌ Cron expression not found in config.json"
        echo "💡 Make sure global.deployment.lambdaCronExpression is set"
        exit 1
    fi

    echo "📅 Using cron expression: $CRON_EXPRESSION"
}

get_aws_account_id() {
    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null)
    if [ $? -ne 0 ] || [ "$AWS_ACCOUNT_ID" = "" ]; then
        echo "❌ Failed to get AWS account ID. Make sure AWS CLI is configured."
        exit 1
    fi
}

get_aws_region() {
    AWS_REGION=$(aws configure get region 2>/dev/null)
    if [ "$AWS_REGION" = "" ]; then
        AWS_REGION="us-east-1"
        echo "⚠️  No default region set, using us-east-1"
    fi
    echo "🌍 Using AWS region: $AWS_REGION"
}

create_iam_role() {
    local role_name="telegraws-${FUNCTION_NAME}-role"

    echo "🔐 Creating IAM role: $role_name"

    # Trust policy for Lambda
    cat >/tmp/trust-policy.json <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "lambda.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF

    # Create role
    aws iam create-role \
        --role-name "$role_name" \
        --assume-role-policy-document file:///tmp/trust-policy.json \
        --description "Role for Telegraws $FUNCTION_NAME Lambda function" >/dev/null

    # Permission policy
    cat >/tmp/lambda-policy.json <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "logs:CreateLogGroup",
            "Resource": "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": [
                "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:log-group:/aws/lambda/telegraws-${FUNCTION_NAME}:*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "wafv2:GetWebACL",
                "wafv2:ListResourcesForWebACL",
                "cloudwatch:GetMetricStatistics",
                "cloudwatch:ListMetrics",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF

    # Create and attach policy
    aws iam put-role-policy \
        --role-name "$role_name" \
        --policy-name "telegraws-${FUNCTION_NAME}-policy" \
        --policy-document file:///tmp/lambda-policy.json

    # Clean up temp files
    rm -f /tmp/trust-policy.json /tmp/lambda-policy.json

    echo "✅ IAM role created: $role_name"

    # Wait a bit for role to be available
    echo "⏳ Waiting for IAM role to be available..."
    sleep 10
}

create_lambda_function() {
    local role_arn="arn:aws:iam::${AWS_ACCOUNT_ID}:role/telegraws-${FUNCTION_NAME}-role"
    local lambda_name="telegraws-${FUNCTION_NAME}"

    echo "🚀 Creating Lambda function: $lambda_name"

    aws lambda create-function \
        --function-name "$lambda_name" \
        --runtime provided.al2023 \
        --role "$role_arn" \
        --handler bootstrap \
        --zip-file "fileb://$BUILD_DIR/${FUNCTION_NAME}.zip" \
        --timeout 120 \
        --architecture arm64 \
        --description "Telegraws monitoring function" >/dev/null

    if [ $? -eq 0 ]; then
        echo "✅ Lambda function created successfully"
    else
        echo "❌ Failed to create Lambda function"
        exit 1
    fi
}

create_eventbridge_schedule() {
    local rule_name="telegraws-${FUNCTION_NAME}-schedule"
    local lambda_name="telegraws-${FUNCTION_NAME}"

    echo "📅 Creating EventBridge schedule: $rule_name"

    # Create EventBridge rule
    aws events put-rule \
        --name "$rule_name" \
        --schedule-expression "cron($CRON_EXPRESSION)" \
        --description "Schedule for Telegraws $FUNCTION_NAME" \
        --state ENABLED >/dev/null

    # Add permission for EventBridge to invoke Lambda
    aws lambda add-permission \
        --function-name "$lambda_name" \
        --statement-id "telegraws-${FUNCTION_NAME}-eventbridge-permission" \
        --action lambda:InvokeFunction \
        --principal events.amazonaws.com \
        --source-arn "arn:aws:events:${AWS_REGION}:${AWS_ACCOUNT_ID}:rule/${rule_name}" >/dev/null

    # Add Lambda as target to the rule
    aws events put-targets \
        --rule "$rule_name" \
        --targets "Id"="1","Arn"="arn:aws:lambda:${AWS_REGION}:${AWS_ACCOUNT_ID}:function:${lambda_name}" >/dev/null

    echo "✅ EventBridge schedule created and linked"
}

update_eventbridge_schedule() {
    local rule_name="telegraws-${FUNCTION_NAME}-schedule"
    local lambda_name="telegraws-${FUNCTION_NAME}"

    echo "📅 Updating EventBridge schedule: $rule_name"

    aws events put-rule \
        --name "$rule_name" \
        --schedule-expression "cron($CRON_EXPRESSION)" \
        --description "Schedule for Telegraws $FUNCTION_NAME" \
        --state ENABLED >/dev/null

    aws events put-targets \
        --rule "$rule_name" \
        --targets "Id"="1","Arn"="arn:aws:lambda:${AWS_REGION}:${AWS_ACCOUNT_ID}:function:${lambda_name}" >/dev/null
}


check_function_exists() {
    aws lambda get-function --function-name "telegraws-$FUNCTION_NAME" >/dev/null 2>&1
    return $?
}

if [ "$MODE" = "local" ]; then
    echo "🏃 Running locally..."
    cd "$SCRIPT_DIR"
    go run .
    exit $?
fi

# Get configuration
get_function_name
get_cron_expression
get_aws_account_id
get_aws_region

# Lambda build
echo "Cleaning build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo "Building for AWS Lambda..."
cd "$SCRIPT_DIR"
GOOS=linux GOARCH=arm64 go build -o "$BUILD_DIR/bootstrap"

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# Create zip file
echo "Creating zip file..."
cd "$BUILD_DIR"
zip "${FUNCTION_NAME}.zip" bootstrap

if [ $? -ne 0 ]; then
    echo "❌ Zip creation failed!"
    exit 1
fi

echo "✅ Lambda build complete. Output: $BUILD_DIR/${FUNCTION_NAME}.zip"

# Check if function exists
if check_function_exists; then
    echo "📦 Lambda function exists, updating code..."
    aws lambda update-function-code \
        --function-name "telegraws-$FUNCTION_NAME" \
        --zip-file "fileb://$BUILD_DIR/${FUNCTION_NAME}.zip"

    if [ $? -eq 0 ]; then
        echo "✅ Lambda function updated successfully!"
    else
        echo "❌ Failed to update Lambda function!"
        exit 1
    fi

    update_eventbridge_schedule
else
    echo "🆕 Lambda function doesn't exist, creating infrastructure..."

    # Create all infrastructure
    create_iam_role
    create_lambda_function
    create_eventbridge_schedule

    echo "🎉 Infrastructure created successfully!"
    echo "📋 Summary:"
    echo "   - Function: telegraws-$FUNCTION_NAME"
    echo "   - Role: telegraws-${FUNCTION_NAME}-role"
    echo "   - Schedule: telegraws-${FUNCTION_NAME}-schedule"
    echo "   - Cron: $CRON_EXPRESSION"
    echo "   - Region: $AWS_REGION"
fi
