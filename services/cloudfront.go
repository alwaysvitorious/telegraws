package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func CloudFrontMetrics(ctx context.Context, cwClient *cloudwatch.Client, distributionID string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	cloudFrontMetrics := []struct {
		Name      string
		Statistic string
		Unit      string
	}{
		{"Requests", "Sum", "Count"},
		{"BytesDownloaded", "Sum", "Bytes"},
		{"4xxErrorRate", "Average", "Percent"},
		{"5xxErrorRate", "Average", "Percent"},
		{"CacheHitRate", "Average", "Percent"},
		{"OriginLatency", "Average", "Milliseconds"},
	}

	for _, metric := range cloudFrontMetrics {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/CloudFront"),
			MetricName: aws.String(metric.Name),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("DistributionId"),
					Value: aws.String(distributionID),
				},
			},
			StartTime:  aws.Time(timeParams["startTime"]),
			EndTime:    aws.Time(timeParams["endTime"]),
			Period:     period,
			Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
		}

		if metric.Unit != "" {
			input.Unit = types.StandardUnit(metric.Unit)
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
				if metric.Name == "BytesDownloaded" || metric.Name == "BytesUploaded" {
					value = value / (1024.0 * 1024.0)
				}
			}
			metrics[metricKey] = value
		} else {
			metrics[metricKey] = 0.0
		}
	}

	return metrics, nil
}
