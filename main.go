package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"telegraws/config"
	"telegraws/services"
	"telegraws/utils"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"go.uber.org/zap"
)

func logic(ctx context.Context) error {
	appConfig, err := config.LoadEmbeddedConfig()
	if err != nil {
		return fmt.Errorf("failed to load app config: %v", err)
	}

	timeParams, err := appConfig.GetTimeParams()
	if err != nil {
		return fmt.Errorf("failed to calculate time parameters: %v", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	cwClient := cloudwatch.NewFromConfig(awsCfg)
	logsClient := cloudwatchlogs.NewFromConfig(awsCfg)
	wafClient := wafv2.NewFromConfig(awsCfg)

	allMetrics := make(map[string]any)

	timeParamsMap := map[string]time.Time{
		"startTime": timeParams.StartTime,
		"endTime":   timeParams.EndTime,
	}

	if appConfig.Services.EC2.Enabled {
		ec2Metrics, err := services.EC2Metrics(ctx, cwClient, appConfig.Services.EC2.InstanceID, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get EC2 metrics", zap.Error(err))
		} else {
			allMetrics["ec2"] = ec2Metrics
		}
	}

	if appConfig.Services.S3.Enabled && timeParams.IsDailyReport {
		s3Metrics, err := services.S3Metrics(ctx, cwClient, appConfig.Services.S3.BucketName, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get S3 metrics", zap.Error(err))
		} else {
			allMetrics["s3"] = s3Metrics
		}
	}

	if appConfig.Services.ALB.Enabled {
		albMetrics, err := services.ALBMetrics(ctx, cwClient, appConfig.Services.ALB.ALBName, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get ALB metrics", zap.Error(err))
		} else {
			allMetrics["alb"] = albMetrics
		}
	}

	if appConfig.Services.CloudFront.Enabled {
		cloudFrontMetrics, err := services.CloudFrontMetrics(ctx, cwClient, appConfig.Services.CloudFront.DistributionID, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get CloudFront metrics", zap.Error(err))
		} else {
			allMetrics["cloudfront"] = cloudFrontMetrics
		}
	}

	if appConfig.Services.CloudWatchAgent.Enabled {
		cwAgentMetrics, err := services.CWAgentMetrics(ctx, cwClient, appConfig.Services.CloudWatchAgent.InstanceID, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get CloudWatch Agent metrics", zap.Error(err))
		} else {
			allMetrics["cloudwatchAgent"] = cwAgentMetrics
		}
	}

	if appConfig.Services.CloudWatchLogs.Enabled {
		logMetrics := make(map[string]any)
		for _, logGroupName := range appConfig.Services.CloudWatchLogs.LogGroupNames {
			logCounts, err := services.CWLogs(ctx, logsClient, logGroupName, timeParamsMap)
			if err != nil {
				utils.Logger.Error("Failed to get CloudWatch Logs metrics",
					zap.Error(err),
					zap.String("logGroup", logGroupName),
				)
				continue
			}
			logMetrics[logGroupName] = logCounts
		}
		if len(logMetrics) > 0 {
			allMetrics["cloudwatchLogs"] = logMetrics
		}
	}

	if appConfig.Services.WAF.Enabled {
		wafMetrics, err := services.WAFMetrics(ctx, wafClient, cwClient, appConfig.Services.WAF.WebACLID, appConfig.Services.WAF.WebACLName, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get WAF metrics", zap.Error(err))
		} else {
			allMetrics["waf"] = wafMetrics
		}

	}

	if appConfig.Services.DynamoDB.Enabled {
		dynamoMetrics := make(map[string]any)
		for _, tableName := range appConfig.Services.DynamoDB.TableNames {
			dynamodbMetrics, err := services.DynamoDBMetrics(ctx, cwClient, timeParamsMap, tableName)
			if err != nil {
				utils.Logger.Error("Failed to get DynamoDB metrics",
					zap.Error(err),
					zap.String("tableName", tableName),
				)
				continue
			}
			dynamoMetrics[tableName] = dynamodbMetrics
		}
		if len(dynamoMetrics) > 0 {
			allMetrics["dynamodb"] = dynamoMetrics
		}
	}

	if appConfig.Services.RDS.Enabled {
		rdsMetrics, err := services.RDSMetrics(ctx, cwClient, appConfig.Services.RDS.ClusterID, appConfig.Services.RDS.DBInstanceIdentifier, timeParamsMap)
		if err != nil {
			utils.Logger.Error("Failed to get RDS metrics", zap.Error(err))
		} else {
			allMetrics["rds"] = rdsMetrics
		}
	}

	message := utils.BuildMessage(appConfig, timeParams, allMetrics, appConfig.Global.Notifications.UseEmail)

	if appConfig.Global.Notifications.UseEmail {
		e := appConfig.Global.Notifications.Email
		if err := utils.SendToEmail(
			e.Host,
			e.Port,
			e.Username,
			e.Password,
			e.HeaderFrom,
			e.EnvelopeFrom,
			e.ToAddr,
			message,
		); err != nil {
			utils.Logger.Error("Failed to send email", zap.Error(err))
			return err
		}
	} else {
		t := appConfig.Global.Notifications.Telegram
		if err := utils.SendToTelegram(ctx, message, t.BotToken, t.ChatID); err != nil {
			utils.Logger.Error("Failed to send Telegram message", zap.Error(err))
			return err
		}
	}

	return nil
}

func main() {
	ctx := context.Background()
	defer utils.Logger.Sync()

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(func(ctx context.Context) error {
			return logic(ctx)
		})
	} else {
		if err := logic(ctx); err != nil {
			log.Printf("Error executing logic: %v", err)
		}
	}
}
