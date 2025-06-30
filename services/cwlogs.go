package services

import (
	"context"
	"telegraws/utils"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"go.uber.org/zap"
)

func CWLogs(ctx context.Context, logsClient *cloudwatchlogs.Client, logGroupName string, timeParams map[string]time.Time) (map[string]int, error) {
	levels := map[string]string{
		"error": "{ $.level = \"error\" }",
		"warn":  "{ $.level = \"warn\" }",
		"info":  "{ $.level = \"info\" }",
	}

	counts := map[string]int{
		"error": 0,
		"warn":  0,
		"info":  0,
	}

	for level, filterPattern := range levels {
		input := &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName:  aws.String(logGroupName),
			FilterPattern: aws.String(filterPattern),
			StartTime:     aws.Int64(timeParams["startTime"].UnixMilli()),
			EndTime:       aws.Int64(timeParams["endTime"].UnixMilli()),
		}

		// Use pagination to get all matching logs
		var count int
		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(logsClient, input)
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				// Don't fail the whole report for log counting issues
				utils.Logger.Error("Failed to count logs",
					zap.Error(err),
					zap.String("level", level),
					zap.String("logGroup", logGroupName),
					zap.String("filterPattern", filterPattern),
				)
				break
			}
			count += len(output.Events)
		}

		counts[level] = count
	}

	return counts, nil
}
