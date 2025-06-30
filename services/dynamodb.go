package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func DynamoDBMetrics(ctx context.Context, cwClient *cloudwatch.Client, timeParams map[string]time.Time, tableName string) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	dynamoMetrics := []struct {
		Name      string
		Statistic string
		Unit      string
	}{
		{"ReadThrottledRequests", "Sum", "count"},
		{"WriteThrottledRequests", "Sum", "count"},
		{"SuccessfulRequestLatency", "Average", "ms"},
		{"SystemErrors", "Sum", "count"},
		{"UserErrors", "Sum", "count"},
		{"ConsumedReadCapacityUnits", "Sum", "count"},
		{"ConsumedWriteCapacityUnits", "Sum", "count"},
		{"RequestCount", "Sum", "count"},
	}

	for _, metric := range dynamoMetrics {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/DynamoDB"),
			MetricName: aws.String(metric.Name),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("TableName"),
					Value: aws.String(tableName),
				},
			},
			StartTime:  aws.Time(timeParams["startTime"]),
			EndTime:    aws.Time(timeParams["endTime"]),
			Period:     period,
			Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
		}

		result, err := cwClient.GetMetricStatistics(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error getting %s: %v", metric.Name, err)
		}

		metricKey := metric.Name

		if len(result.Datapoints) > 0 {
			var value float64
			switch metric.Statistic {
			case "Average":
				value = *result.Datapoints[0].Average
			case "Sum":
				value = *result.Datapoints[0].Sum
			}
			metrics[metricKey] = value
		} else {
			metrics[metricKey] = 0.0
		}
	}

	return metrics, nil
}
