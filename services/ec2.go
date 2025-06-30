package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Does NOT track disk read/write metrics (EBS volumes)

func EC2Metrics(ctx context.Context, cwClient *cloudwatch.Client, instanceID string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	ec2Metrics := []struct {
		Name      string
		Statistic string
		Unit      string
	}{
		{"CPUUtilization", "Average", "%"},
		{"CPUUtilization", "Maximum", "%"},
		{"StatusCheckFailed", "Sum", "count"},
		{"NetworkIn", "Sum", "MB"},
		{"NetworkOut", "Sum", "MB"},
	}

	for _, metric := range ec2Metrics {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/EC2"),
			MetricName: aws.String(metric.Name),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("InstanceId"),
					Value: aws.String(instanceID),
				},
			},
			StartTime:  aws.Time(timeParams["startTime"]),
			EndTime:    aws.Time(timeParams["endTime"]),
			Period:     period,
			Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
		}

		if metric.Name == "NetworkIn" || metric.Name == "NetworkOut" {
			input.Unit = types.StandardUnit("Bytes")
		}

		result, err := cwClient.GetMetricStatistics(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error getting %s: %v", metric.Name, err)
		}

		metricKey := metric.Name
		if metric.Name == "CPUUtilization" {
			metricKey = fmt.Sprintf("%s_%s", metric.Name, metric.Statistic)
		}

		// Process based on statistic type
		if len(result.Datapoints) > 0 {
			var value float64
			switch metric.Statistic {
			case "Average":
				value = *result.Datapoints[0].Average
			case "Maximum":
				value = *result.Datapoints[0].Maximum
			case "Sum":
				value = *result.Datapoints[0].Sum
				if metric.Name == "NetworkIn" || metric.Name == "NetworkOut" {
					value = value / (1024.0 * 1024.0) // Convert to MB
				}
			}
			metrics[metricKey] = value
		} else {
			metrics[metricKey] = 0.0
		}
	}

	return metrics, nil
}
