package services

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func S3Metrics(ctx context.Context, cwClient *cloudwatch.Client, bucketName string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(86400) // Always use daily for S3

	// Storage metrics (daily reporting)
	storageMetrics := []struct {
		Name         string
		Statistic    string
		StorageTypes []string // Try multiple storage types
	}{
		{"BucketSizeBytes", "Average", []string{
			"StandardStorage",
			"StandardIAStorage",
			"StandardIASizeOverhead",
			"ReducedRedundancyStorage",
			"GlacierStorage",
			"DeepArchiveStorage",
			"GlacierInstantRetrievalSizeOverhead",
			"GlacierFlexibleRetrievalSizeOverhead",
			"GlacierDeepArchiveSizeOverhead",
			"IntelligentTieringFAStorage",
			"IntelligentTieringIAStorage",
			"IntelligentTieringAAStorage",
			"IntelligentTieringAIAStorage",
			"IntelligentTieringDAAStorage",
		}},
	}

	// Try storage metrics with different storage types
	for _, metric := range storageMetrics {
		found := false
		for _, storageType := range metric.StorageTypes {
			dimensions := []types.Dimension{
				{
					Name:  aws.String("BucketName"),
					Value: aws.String(bucketName),
				},
				{
					Name:  aws.String("StorageType"),
					Value: aws.String(storageType),
				},
			}

			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/S3"),
				MetricName: aws.String(metric.Name),
				Dimensions: dimensions,
				StartTime:  aws.Time(timeParams["startTime"]),
				EndTime:    aws.Time(timeParams["endTime"]),
				Period:     period,
				Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
			}

			result, err := cwClient.GetMetricStatistics(ctx, input)
			if err != nil {
				continue // Try next storage type
			}

			if len(result.Datapoints) > 0 && result.Datapoints[0].Average != nil {
				value := *result.Datapoints[0].Average

				if metric.Name == "BucketSizeBytes" {
					value = value / (1024.0 * 1024.0)
				}

				metrics[metric.Name] = value
				found = true
				break // Found data, stop trying other storage types
			}
		}

		if !found {
			metrics[metric.Name] = 0.0
		}
	}

	// Request metrics (only if enabled in bucket - these are often 0)
	requestMetrics := []struct {
		Name      string
		Statistic string
	}{
		{"AllRequests", "Sum"},
		{"4xxErrors", "Sum"},
		{"5xxErrors", "Sum"},
	}

	// Use hourly period for request metrics
	requestPeriod := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		requestPeriod = aws.Int32(86400)
	}

	for _, metric := range requestMetrics {
		dimensions := []types.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
		}

		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/S3"),
			MetricName: aws.String(metric.Name),
			Dimensions: dimensions,
			StartTime:  aws.Time(timeParams["startTime"]),
			EndTime:    aws.Time(timeParams["endTime"]),
			Period:     requestPeriod,
			Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
		}

		result, err := cwClient.GetMetricStatistics(ctx, input)
		if err != nil {
			metrics[metric.Name] = 0.0
			continue
		}

		if len(result.Datapoints) > 0 && result.Datapoints[0].Sum != nil {
			metrics[metric.Name] = *result.Datapoints[0].Sum
		} else {
			metrics[metric.Name] = 0.0
		}
	}

	return metrics, nil
}
