# Telegraws

**AWS Infrastructure Monitoring with Telegram & Email (SMTP) Notifications**

Telegraws is a lightweight, serverless AWS monitoring Lambda that collects
metrics from multiple AWS services and sends automated reports to **Telegram or
Email**.

## Features

- **Serverless**: Deploys as AWS Lambda function with automatic scheduling.
- **Telegram Integration**: Sends formatted monitoring reports to Telegram.
- **Email Integration (SMTP)**: Optionally deliver the same reports via SMTP
  email.
- **IAC**: Automatically creates IAM roles, Lambda functions, and EventBridge
  schedules.
- **Local Development**: Test locally with `--local` flag before deployment.
- **Multi-Service Monitoring**: EC2, S3, ALB, CloudFront, DynamoDB, RDS, WAF,
  CloudWatch Logs, Cloudwatch Agents.
- **Smart Scheduling**: Hourly updates + daily reports.
- **Immutable Deployments**: Clean, reproducible deployments.

## Prerequisites

- **AWS CLI** installed and configured in the same region as your monitoring
  infrastructure.
- **Go 1.24+** installed.
- **Telegram Bot** token and chat ID (if using Telegram).
- **SMTP credentials** (host, port, username, password, from, to) if using email
  notifications.
- **AWS Resources** to monitor.

## Dependencies

```
github.com/aws/aws-lambda-go/lambda
github.com/aws/aws-sdk-go-v2
go.uber.org/zap
```

## Usage

```bash
git clone https://github.com/alwaysvitorious/telegraws.git
cd telegraws
go mod tidy
cp config/config-template.json config/config.json
# Edit config/config.json with your settings
./build.sh --lambda # or --local
```

## Considerations

- When running ./build.sh --lambda, it automatically detects if the function was
  already deployed and updates the existing function.
- Telegraws creates these resources: Lambda function, IAM role, Eventbridge
  rule. All prefixed by 'telegraws-'.
- lambdaCronExpression: EventBridge cron schedule (AWS format: Minutes Hours Day
  Month DayOfWeek Year).
- defaultPeriod: Hours to look back for regular reports (1 = last hour).
- dailyReportHourUTC: Hour to send daily summary.
- CloudWatch Logs collection counts INFO/WARN/ERROR so structured logging is
  required.
- RDS monitoring currently supports Aurora engine.
- WAF monitoring collects WAFs metrics attached to ALB.
- Some S3 metrics require S3 request metrics to be enabled.
- CloudWatch Agent monitors disk_used_percent and mem_used_percent.
- Telegram has 4096 character limit per message.

## Metrics

- EC2: CPU Utilization (avg/max), Network I/O, Status Checks. If CloudWatch
  Agent: mem_used_percent, disk_used_percent.

- S3: (Daily Reports Only) Bucket Size, Request Count, Error Rates.

- ALB: Request Count, Response Time, HTTP Status Codes, Healthy/Unhealthy Hosts,
  ALB Errors.

- CloudFront: Requests, Data Downloaded, Cache Hit Rate, Error Rates, Origin
  Latency.

- DynamoDB: Request Count, Throttles, Latency, Consumed Capacity, Error Counts.

- RDS/Aurora: Instance: CPU, Memory, Connections, Read/Write Latency. Cluster:
  Volume Size, IOPS.

- WAF: Allowed/Blocked Requests.

- CloudWatch Logs: INFO/WARN/ERROR log counts (requires structured logging).

## To-do

- Enhanced Metrics: Add comprehensive metric collection for all services. Get
  metrics dynamically using AWS CLI?
- Dynamic Metrics: User-configurable metrics selection. Separated daily and
  scheduled metrics.
- Enhanced AWS Support: ECS/EKS, Fargate, API Gateway.
- Multi-Resource: Multiple IDs per service type.
- Message Splitting: Handle Telegram 4096 character limit.
- Environment Variables: Support for .env configuration.
- Cross-Platform: Windows support for build script.
- Emoji Support: Optional emoji integration in messages.
- Architecture Options: x86_64 Lambda support.
- RDS Engines: Support for MySQL, PostgreSQL, SQL Server.
- Advanced WAF: Multiple WAF configurations.

## License

This project is fully open source and available under the MIT License.

Contributions are welcome! Please feel free to submit a Pull Request.
