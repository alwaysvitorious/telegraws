package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func ALBMetrics(ctx context.Context, cwClient *cloudwatch.Client, albName string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	// If albName doesn't start with "app/", assume it's just the name and we need to find the full identifier
	var loadBalancerDimension string
	if strings.HasPrefix(albName, "app/") {
		// Already the full LoadBalancer identifier
		loadBalancerDimension = albName
	} else {
		// Need to find the full identifier by listing metrics
		listInput := &cloudwatch.ListMetricsInput{
			Namespace:  aws.String("AWS/ApplicationELB"),
			MetricName: aws.String("RequestCount"),
		}

		listResult, err := cwClient.ListMetrics(ctx, listInput)
		if err != nil {
			return nil, fmt.Errorf("error listing ALB metrics: %v", err)
		}

		// Find the LoadBalancer dimension that contains our ALB name
		for _, metric := range listResult.Metrics {
			for _, dimension := range metric.Dimensions {
				if *dimension.Name == "LoadBalancer" &&
					strings.Contains(*dimension.Value, albName) {
					loadBalancerDimension = *dimension.Value
					break
				}
			}
			if loadBalancerDimension != "" {
				break
			}
		}

		if loadBalancerDimension == "" {
			return nil, fmt.Errorf("could not find LoadBalancer dimension for ALB: %s", albName)
		}
	}

	albMetrics := []struct {
		Name      string
		Statistic string
		Unit      string
	}{
		{"RequestCount", "Sum", "Count"},
		{"TargetResponseTime", "Average", "Seconds"},
		{"HTTPCode_Target_2XX_Count", "Sum", "Count"},
		{"HTTPCode_Target_4XX_Count", "Sum", "Count"},
		{"HTTPCode_Target_5XX_Count", "Sum", "Count"},
		{"HTTPCode_ELB_4XX_Count", "Sum", "Count"},
		{"HTTPCode_ELB_5XX_Count", "Sum", "Count"},
		{"HealthyHostCount", "Average", "Count"},
		{"UnHealthyHostCount", "Average", "Count"},
	}

	for _, metric := range albMetrics {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/ApplicationELB"),
			MetricName: aws.String(metric.Name),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("LoadBalancer"),
					Value: aws.String(loadBalancerDimension),
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
			}
			metrics[metricKey] = value
		} else {
			metrics[metricKey] = 0.0
		}
	}

	return metrics, nil
}
