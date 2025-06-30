package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func CWAgentMetrics(ctx context.Context, cwClient *cloudwatch.Client, instanceID string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	// Memory metrics (average and maximum)
	memMetrics := []string{"Average", "Maximum"}
	for _, stat := range memMetrics {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("CWAgent"),
			MetricName: aws.String("mem_used_percent"),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("InstanceId"),
					Value: aws.String(instanceID),
				},
			},
			StartTime:  aws.Time(timeParams["startTime"]),
			EndTime:    aws.Time(timeParams["endTime"]),
			Period:     period,
			Statistics: []types.Statistic{types.Statistic(stat)},
		}

		result, err := cwClient.GetMetricStatistics(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error getting mem_used_percent (%s): %v", stat, err)
		}

		metricKey := fmt.Sprintf("mem_used_percent_%s", stat)
		if len(result.Datapoints) > 0 {
			if stat == "Average" {
				metrics[metricKey] = *result.Datapoints[0].Average
			} else {
				metrics[metricKey] = *result.Datapoints[0].Maximum
			}
		} else {
			metrics[metricKey] = 0.0
		}
	}

	// Disk metrics (with proper dimensions)
	// First, discover the device and fstype dimensions
	listInput := &cloudwatch.ListMetricsInput{
		Namespace:  aws.String("CWAgent"),
		MetricName: aws.String("disk_used_percent"),
		Dimensions: []types.DimensionFilter{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
			{
				Name:  aws.String("path"),
				Value: aws.String("/"),
			},
		},
	}

	listResult, err := cwClient.ListMetrics(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("error listing disk metrics: %v", err)
	}

	var device, fstype string
	for _, metric := range listResult.Metrics {
		isCorrectInstance := false
		for _, dim := range metric.Dimensions {
			if *dim.Name == "InstanceId" && *dim.Value == instanceID {
				isCorrectInstance = true
				break
			}
		}

		if !isCorrectInstance {
			continue
		}

		for _, dim := range metric.Dimensions {
			if dim.Name == nil {
				continue
			}

			switch *dim.Name {
			case "device":
				if dim.Value != nil {
					device = *dim.Value
				}
			case "fstype":
				if dim.Value != nil {
					fstype = *dim.Value
				}
			}
		}

		if device != "" && fstype != "" {
			break
		}
	}

	// Get disk_used_percent metric with the discovered dimensions
	diskInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("CWAgent"),
		MetricName: aws.String("disk_used_percent"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
			{
				Name:  aws.String("path"),
				Value: aws.String("/"),
			},
			{
				Name:  aws.String("device"),
				Value: aws.String(device),
			},
			{
				Name:  aws.String("fstype"),
				Value: aws.String(fstype),
			},
		},
		StartTime:  aws.Time(timeParams["startTime"]),
		EndTime:    aws.Time(timeParams["endTime"]),
		Period:     period,
		Statistics: []types.Statistic{types.Statistic("Average")},
	}

	diskResult, err := cwClient.GetMetricStatistics(ctx, diskInput)
	if err != nil {
		return nil, fmt.Errorf("error getting disk_used_percent: %v", err)
	}

	if len(diskResult.Datapoints) > 0 {
		metrics["disk_used_percent"] = *diskResult.Datapoints[0].Average
	} else {
		metrics["disk_used_percent"] = 0.0
	}

	return metrics, nil
}
