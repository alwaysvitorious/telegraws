package utils

import (
	"fmt"
	"strings"
	"telegraws/config"
)

// Helper function to escape Telegram markdown characters
func escapeMarkdown(text string) string {
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "*", "\\*")
	return text
}

func BuildMessage(cfg *config.Config, timeParams *config.TimeParams, allMetrics map[string]any) string {
	messageBuilder := strings.Builder{}

	scheduleSeparator := "- - - - - - - - - - - - - - -"
	dailySeparator := "* * * * * * * * * * * * * * *"

	if timeParams.IsDailyReport {
		messageBuilder.WriteString("\n" + dailySeparator + "\n\n")
	} else {
		messageBuilder.WriteString("\n" + scheduleSeparator + "\n\n")
	}

	messageBuilder.WriteString(fmt.Sprintf("%s\n\n", timeParams.EndTime.Format("02/01/2006 15:04:05")))

	if cfg.Services.EC2.Enabled {
		if ec2Data, exists := allMetrics["ec2"]; exists {
			ec2Metrics := ec2Data.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("*EC2*: %s\n", cfg.Services.EC2.InstanceID))
			messageBuilder.WriteString(fmt.Sprintf("CPU: %.2f%% (avg), %.2f%% (max)\n",
				ec2Metrics["CPUUtilization_Average"],
				ec2Metrics["CPUUtilization_Maximum"]))
			messageBuilder.WriteString(fmt.Sprintf("Status Checks Failed: %.0f\n", ec2Metrics["StatusCheckFailed"]))
			messageBuilder.WriteString(fmt.Sprintf("Network In: %.2f MB\n", ec2Metrics["NetworkIn"]))
			messageBuilder.WriteString(fmt.Sprintf("Network Out: %.2f MB\n", ec2Metrics["NetworkOut"]))
		}
	}

	if cfg.Services.CloudWatchAgent.Enabled {
		if cwAgentData, exists := allMetrics["cloudwatchAgent"]; exists {
			cwAgentMetrics := cwAgentData.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("Memory: %.2f%% (avg), %.2f%% (max)\n",
				cwAgentMetrics["mem_used_percent_Average"],
				cwAgentMetrics["mem_used_percent_Maximum"]))
			messageBuilder.WriteString(fmt.Sprintf("Disk: %.2f%%\n",
				cwAgentMetrics["disk_used_percent"]))
			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.S3.Enabled && timeParams.IsDailyReport {
		if s3Data, exists := allMetrics["s3"]; exists {
			s3Metrics := s3Data.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("*S3* %s\n", escapeMarkdown(cfg.Services.S3.BucketName)))
			messageBuilder.WriteString(fmt.Sprintf("Size: %.2f MB\n", s3Metrics["BucketSizeBytes"]))
			messageBuilder.WriteString(fmt.Sprintf("Requests: %.0f\n", s3Metrics["AllRequests"]))
			messageBuilder.WriteString(fmt.Sprintf("4xx Errors: %.0f\n", s3Metrics["4xxErrors"]))
			messageBuilder.WriteString(fmt.Sprintf("5xx Errors: %.0f\n", s3Metrics["5xxErrors"]))
			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.ALB.Enabled {
		if albData, exists := allMetrics["alb"]; exists {
			albMetrics := albData.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("*ALB* %s\n", escapeMarkdown(cfg.Services.ALB.ALBName)))
			messageBuilder.WriteString(fmt.Sprintf("Requests: %.0f\n", albMetrics["RequestCount"]))
			messageBuilder.WriteString(fmt.Sprintf("Response Time: %.3f s\n", albMetrics["TargetResponseTime"]))
			messageBuilder.WriteString(fmt.Sprintf("2xx: %.0f, 4xx: %.0f, 5xx: %.0f\n",
				albMetrics["HTTPCode_Target_2XX_Count"],
				albMetrics["HTTPCode_Target_4XX_Count"],
				albMetrics["HTTPCode_Target_5XX_Count"]))

			messageBuilder.WriteString(fmt.Sprintf("Healthy: %.0f, Unhealthy: %.0f\n",
				albMetrics["HealthyHostCount"],
				albMetrics["UnHealthyHostCount"]))

			elbErrors := albMetrics["HTTPCode_ELB_4XX_Count"] + albMetrics["HTTPCode_ELB_5XX_Count"]
			messageBuilder.WriteString(fmt.Sprintf("ALB Errors: %.0f\n", elbErrors))

			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.CloudFront.Enabled {
		if cfData, exists := allMetrics["cloudfront"]; exists {
			cfMetrics := cfData.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("*CloudFront* %s\n", cfg.Services.CloudFront.DistributionID))
			messageBuilder.WriteString(fmt.Sprintf("Requests: %.0f\n", cfMetrics["Requests"]))
			messageBuilder.WriteString(fmt.Sprintf("Data Downloaded: %.2f MB\n", cfMetrics["BytesDownloaded"])) // ← ADD THIS
			messageBuilder.WriteString(fmt.Sprintf("Cache Hit Rate: %.2f%%\n", cfMetrics["CacheHitRate"]))
			messageBuilder.WriteString(fmt.Sprintf("4xx Error Rate: %.2f%%\n", cfMetrics["4xxErrorRate"]))
			messageBuilder.WriteString(fmt.Sprintf("5xx Error Rate: %.2f%%\n", cfMetrics["5xxErrorRate"]))
			messageBuilder.WriteString(fmt.Sprintf("Origin Latency: %.2f ms\n", cfMetrics["OriginLatency"]))
			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.DynamoDB.Enabled {
		if dynamoData, exists := allMetrics["dynamodb"]; exists {
			dynamoMetrics := dynamoData.(map[string]any)
			for _, tableName := range cfg.Services.DynamoDB.TableNames {
				if tableData, tableExists := dynamoMetrics[tableName]; tableExists {
					tableMetrics := tableData.(map[string]float64)

					messageBuilder.WriteString(fmt.Sprintf("*DynamoDB* %s\n", escapeMarkdown(tableName)))
					messageBuilder.WriteString(fmt.Sprintf("Total Requests: %.0f\n", tableMetrics["RequestCount"]))

					messageBuilder.WriteString(fmt.Sprintf("Read Throttles: %.0f\n", tableMetrics["ReadThrottledRequests"]))
					messageBuilder.WriteString(fmt.Sprintf("Write Throttles: %.0f\n", tableMetrics["WriteThrottledRequests"]))
					messageBuilder.WriteString(fmt.Sprintf("Latency: %.2f ms\n", tableMetrics["SuccessfulRequestLatency"]))
					messageBuilder.WriteString(fmt.Sprintf("Read Capacity: %.0f units\n", tableMetrics["ConsumedReadCapacityUnits"]))
					messageBuilder.WriteString(fmt.Sprintf("Write Capacity: %.0f units\n", tableMetrics["ConsumedWriteCapacityUnits"]))

					totalErrors := tableMetrics["UserErrors"] + tableMetrics["SystemErrors"]
					messageBuilder.WriteString(fmt.Sprintf("DB Errors: %.0f\n", totalErrors))
					messageBuilder.WriteString("\n")
				}
			}
		}
	}

	if cfg.Services.RDS.Enabled {
		if rdsData, exists := allMetrics["rds"]; exists {
			rdsMetrics := rdsData.(map[string]float64)

			var rdsHeader string
			if cfg.Services.RDS.ClusterID != "" && cfg.Services.RDS.DBInstanceIdentifier != "" {
				rdsHeader = fmt.Sprintf("*RDS* %s / %s",
					escapeMarkdown(cfg.Services.RDS.ClusterID),
					escapeMarkdown(cfg.Services.RDS.DBInstanceIdentifier))
			} else if cfg.Services.RDS.ClusterID != "" {
				rdsHeader = fmt.Sprintf("*RDS Cluster* %s", escapeMarkdown(cfg.Services.RDS.ClusterID))
			} else {
				rdsHeader = fmt.Sprintf("*RDS Instance* %s", escapeMarkdown(cfg.Services.RDS.DBInstanceIdentifier))
			}

			messageBuilder.WriteString(fmt.Sprintf("%s\n", rdsHeader))

			if cfg.Services.RDS.DBInstanceIdentifier != "" {
				if cpu, exists := rdsMetrics["Instance_CPUUtilization_Average"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("CPU: %.2f%% (avg)", cpu))
					if cpuMax, maxExists := rdsMetrics["Instance_CPUUtilization_Maximum"]; maxExists {
						messageBuilder.WriteString(fmt.Sprintf(", %.2f%% (max)", cpuMax))
					}
					messageBuilder.WriteString("\n")
				}
				if mem, exists := rdsMetrics["Instance_FreeableMemory"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Free Memory: %.2f GB\n", mem))
				}
				if conn, exists := rdsMetrics["Instance_DatabaseConnections"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Connections: %.0f\n", conn))
				}
				if readLat, exists := rdsMetrics["Instance_ReadLatency"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Read Latency: %.2f ms\n", readLat))
				}
				if writeLat, exists := rdsMetrics["Instance_WriteLatency"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Write Latency: %.2f ms\n", writeLat))
				}
			}

			// Show cluster metrics if available
			if cfg.Services.RDS.ClusterID != "" {
				if volume, exists := rdsMetrics["Cluster_VolumeBytesUsed"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Volume Size: %.2f GB\n", volume))
				}
				if readIOPS, exists := rdsMetrics["Cluster_VolumeReadIOPs"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Read IOPS: %.0f\n", readIOPS))
				}
				if writeIOPS, exists := rdsMetrics["Cluster_VolumeWriteIOPs"]; exists {
					messageBuilder.WriteString(fmt.Sprintf("Write IOPS: %.0f\n", writeIOPS))
				}
			}

			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.WAF.Enabled {
		if wafData, exists := allMetrics["waf"]; exists {
			wafMetrics := wafData.(map[string]float64)
			messageBuilder.WriteString(fmt.Sprintf("*WAF* %s\n", escapeMarkdown(cfg.Services.WAF.WebACLName)))
			messageBuilder.WriteString(fmt.Sprintf("Allowed Requests: %.0f\n", wafMetrics["AllowedRequests"]))
			messageBuilder.WriteString(fmt.Sprintf("Blocked Requests: %.0f\n", wafMetrics["BlockedRequests"]))
			messageBuilder.WriteString("\n")
		}
	}

	if cfg.Services.CloudWatchLogs.Enabled {
		if logsData, exists := allMetrics["cloudwatchLogs"]; exists {
			logsMetrics := logsData.(map[string]any)

			applicationLogs := make(map[string]any)
			lambdaLogs := make(map[string]any)

			for _, logGroupName := range cfg.Services.CloudWatchLogs.LogGroupNames {
				if logData, logExists := logsMetrics[logGroupName]; logExists {
					if strings.Contains(logGroupName, "/aws/lambda/") {
						lambdaLogs[logGroupName] = logData
					} else {
						applicationLogs[logGroupName] = logData
					}
				}
			}

			if len(applicationLogs) > 0 {
				messageBuilder.WriteString("*APPLICATION*\n")
				for logGroup, logData := range applicationLogs {
					logCounts := logData.(map[string]int)
					messageBuilder.WriteString(fmt.Sprintf("%s:\n", escapeMarkdown(logGroup)))
					messageBuilder.WriteString(fmt.Sprintf("INFO: %d\n", logCounts["info"]))
					messageBuilder.WriteString(fmt.Sprintf("WARN: %d\n", logCounts["warn"]))
					messageBuilder.WriteString(fmt.Sprintf("ERROR: %d\n", logCounts["error"]))
					messageBuilder.WriteString("\n")
				}
			}

			if len(lambdaLogs) > 0 {
				messageBuilder.WriteString("*LAMBDA*\n")
				for logGroup, logData := range lambdaLogs {
					logCounts := logData.(map[string]int)
					messageBuilder.WriteString(fmt.Sprintf("%s:\n", escapeMarkdown(logGroup)))
					messageBuilder.WriteString(fmt.Sprintf("INFO: %d\n", logCounts["info"]))
					messageBuilder.WriteString(fmt.Sprintf("WARN: %d\n", logCounts["warn"]))
					messageBuilder.WriteString(fmt.Sprintf("ERROR: %d\n", logCounts["error"]))
					messageBuilder.WriteString("\n")
				}
			}
		}
	}

	if timeParams.IsDailyReport {
		messageBuilder.WriteString(dailySeparator + "\n")
	} else {
		messageBuilder.WriteString(scheduleSeparator + "\n")
	}

	return messageBuilder.String()
}
