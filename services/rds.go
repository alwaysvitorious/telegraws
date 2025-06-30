package services

import (
	"context"
	"fmt"
	"strings"
	"telegraws/utils"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"go.uber.org/zap"
)

func RDSMetrics(ctx context.Context, cwClient *cloudwatch.Client, clusterID string, instanceID string, timeParams map[string]time.Time) (map[string]float64, error) {
	metrics := map[string]float64{}
	period := aws.Int32(3600)
	if timeParams["endTime"].Sub(timeParams["startTime"]) >= 24*time.Hour {
		period = aws.Int32(86400)
	}

	if clusterID == "" && instanceID == "" {
		return nil, fmt.Errorf("both clusterID and instanceID are empty - at least one is required")
	}

	// Instance-level metrics (per database instance)
	if instanceID != "" {

		instanceMetrics := []struct {
			Name      string
			Statistic string
			Unit      string
		}{
			{"CPUUtilization", "Average", "%"},
			{"CPUUtilization", "Maximum", "%"},
			{"FreeableMemory", "Average", "bytes"},
			{"DatabaseConnections", "Maximum", "count"},
			{"ReadLatency", "Average", "seconds"},
			{"WriteLatency", "Average", "seconds"},
		}

		for _, metric := range instanceMetrics {
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/RDS"),
				MetricName: aws.String(metric.Name),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("DBInstanceIdentifier"),
						Value: aws.String(instanceID),
					},
				},
				StartTime:  aws.Time(timeParams["startTime"]),
				EndTime:    aws.Time(timeParams["endTime"]),
				Period:     period,
				Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
			}

			result, err := cwClient.GetMetricStatistics(ctx, input)
			if err != nil {
				utils.Logger.Error("Failed to get Aurora instance metric",
					zap.Error(err),
					zap.String("metricName", metric.Name),
					zap.String("statistic", metric.Statistic),
					zap.String("clusterID", clusterID),
					zap.Int32("period", *period),
				)
				continue
			}

			metricKey := fmt.Sprintf("Instance_%s", metric.Name)
			if metric.Name == "CPUUtilization" {
				metricKey = fmt.Sprintf("Instance_CPUUtilization_%s", metric.Statistic)
			}

			if len(result.Datapoints) > 0 {
				var value float64
				switch metric.Statistic {
				case "Average":
					value = *result.Datapoints[0].Average
				case "Maximum":
					value = *result.Datapoints[0].Maximum
				case "Sum":
					value = *result.Datapoints[0].Sum
				}

				if metric.Name == "FreeableMemory" {
					value = value / (1024.0 * 1024.0 * 1024.0)
				}

				if metric.Name == "ReadLatency" || metric.Name == "WriteLatency" {
					value = value * 1000.0
				}

				metrics[metricKey] = value
			} else {
				metrics[metricKey] = 0.0
			}
		}
	}

	// Cluster-level metrics (for the entire Aurora cluster)
	if clusterID != "" {

		clusterMetrics := []struct {
			Name      string
			Statistic string
			Unit      string
		}{
			{"VolumeBytesUsed", "Average", "bytes"},
			{"VolumeReadIOPs", "Average", "count/5min"},
			{"VolumeWriteIOPs", "Average", "count/5min"},
		}

		for _, metric := range clusterMetrics {
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/RDS"),
				MetricName: aws.String(metric.Name),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("DBClusterIdentifier"),
						Value: aws.String(clusterID),
					},
				},
				StartTime:  aws.Time(timeParams["startTime"]),
				EndTime:    aws.Time(timeParams["endTime"]),
				Period:     period,
				Statistics: []types.Statistic{types.Statistic(metric.Statistic)},
			}

			result, err := cwClient.GetMetricStatistics(ctx, input)
			if err != nil {
				utils.Logger.Error("Failed to get Aurora cluster metric",
					zap.Error(err),
					zap.String("metricName", metric.Name),
					zap.String("statistic", metric.Statistic),
					zap.String("clusterID", clusterID),
					zap.Int32("period", *period),
				)
				continue
			}

			metricKey := fmt.Sprintf("Cluster_%s", metric.Name)

			if len(result.Datapoints) > 0 {
				var value float64
				switch metric.Statistic {
				case "Average":
					value = *result.Datapoints[0].Average
				case "Maximum":
					value = *result.Datapoints[0].Maximum
				}

				if strings.Contains(metric.Name, "Storage") || metric.Name == "VolumeBytesUsed" {
					value = value / (1024.0 * 1024.0 * 1024.0)
				}

				if strings.Contains(metric.Name, "Throughput") {
					value = value / (1024.0 * 1024.0)
				}

				metrics[metricKey] = value
			} else {
				metrics[metricKey] = 0.0
			}
		}
	}

	return metrics, nil
}
