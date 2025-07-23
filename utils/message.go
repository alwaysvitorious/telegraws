package utils

import (
	"fmt"
	"html"
	"strings"
	"telegraws/config"
)

func BuildMessage(cfg *config.Config, timeParams *config.TimeParams, allMetrics map[string]any, forEmail bool) string {
	escapeMarkdown := func(text string) string {
		text = strings.ReplaceAll(text, "_", "\\_")
		text = strings.ReplaceAll(text, "*", "\\*")
		return text
	}

	type renderer struct {
		bold func(string) string
		esc  func(string) string
		nl   string
		sep  func(daily bool) string
	}

	tg := renderer{
		bold: func(s string) string { return "*" + s + "*" },
		esc:  escapeMarkdown,
		nl:   "\n",
		sep: func(daily bool) string {
			if daily {
				return "\n= = = = = = = = = = = = = = =\n\n"
			}
			return "\n- - - - - - - - - - - - - - -\n\n"
		},
	}

	htmlR := renderer{
		bold: func(s string) string { return "<strong>" + html.EscapeString(s) + "</strong>" },
		esc:  html.EscapeString,
		nl:   "<br>",
		sep:  func(_ bool) string { return `<hr style="border:none;border-top:1px solid #ccc;margin:12px 0;">` },
	}

	r := tg
	if forEmail {
		r = htmlR
	}

	var b strings.Builder

	// Header
	b.WriteString(r.sep(timeParams.IsDailyReport))
	b.WriteString(timeParams.EndTime.Format("02/01/2006 15:04:05"))
	b.WriteString(r.nl)
	b.WriteString(r.nl)

	// EC2
	if cfg.Services.EC2.Enabled {
		if d, ok := allMetrics["ec2"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("%s: %s%s",
				r.bold("EC2"), r.esc(cfg.Services.EC2.InstanceID), r.nl))
			b.WriteString(fmt.Sprintf("CPU: %.2f%% (avg), %.2f%% (max)%s",
				m["CPUUtilization_Average"], m["CPUUtilization_Maximum"], r.nl))
			b.WriteString(fmt.Sprintf("Status Checks Failed: %.0f%s", m["StatusCheckFailed"], r.nl))
			b.WriteString(fmt.Sprintf("Network In: %.2f MB%s", m["NetworkIn"], r.nl))
			b.WriteString(fmt.Sprintf("Network Out: %.2f MB%s", m["NetworkOut"], r.nl))
		}
	}

	// CloudWatch Agent
	if cfg.Services.CloudWatchAgent.Enabled {
		if d, ok := allMetrics["cloudwatchAgent"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("Memory: %.2f%% (avg), %.2f%% (max)%s",
				m["mem_used_percent_Average"], m["mem_used_percent_Maximum"], r.nl))
			b.WriteString(fmt.Sprintf("Disk: %.2f%%%s", m["disk_used_percent"], r.nl))
			b.WriteString(r.nl)
		}
	}

	// S3 (daily only)
	if cfg.Services.S3.Enabled && timeParams.IsDailyReport {
		if d, ok := allMetrics["s3"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("%s %s%s", r.bold("S3"), r.esc(cfg.Services.S3.BucketName), r.nl))
			b.WriteString(fmt.Sprintf("Size: %.2f MB%s", m["BucketSizeBytes"], r.nl))
			b.WriteString(fmt.Sprintf("Requests: %.0f%s", m["AllRequests"], r.nl))
			b.WriteString(fmt.Sprintf("4xx Errors: %.0f%s", m["4xxErrors"], r.nl))
			b.WriteString(fmt.Sprintf("5xx Errors: %.0f%s", m["5xxErrors"], r.nl))
			b.WriteString(r.nl)
		}
	}

	// ALB
	if cfg.Services.ALB.Enabled {
		if d, ok := allMetrics["alb"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("%s %s%s", r.bold("ALB"), r.esc(cfg.Services.ALB.ALBName), r.nl))
			b.WriteString(fmt.Sprintf("Requests: %.0f%s", m["RequestCount"], r.nl))
			b.WriteString(fmt.Sprintf("Response Time: %.3f s%s", m["TargetResponseTime"], r.nl))
			b.WriteString(fmt.Sprintf("2xx: %.0f, 4xx: %.0f, 5xx: %.0f%s",
				m["HTTPCode_Target_2XX_Count"], m["HTTPCode_Target_4XX_Count"], m["HTTPCode_Target_5XX_Count"], r.nl))
			b.WriteString(fmt.Sprintf("Healthy: %.0f, Unhealthy: %.0f%s",
				m["HealthyHostCount"], m["UnHealthyHostCount"], r.nl))
			elbErrors := m["HTTPCode_ELB_4XX_Count"] + m["HTTPCode_ELB_5XX_Count"]
			b.WriteString(fmt.Sprintf("ALB Errors: %.0f%s", elbErrors, r.nl))
			b.WriteString(r.nl)
		}
	}

	// CloudFront
	if cfg.Services.CloudFront.Enabled {
		if d, ok := allMetrics["cloudfront"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("%s %s%s", r.bold("CloudFront"), r.esc(cfg.Services.CloudFront.DistributionID), r.nl))
			b.WriteString(fmt.Sprintf("Requests: %.0f%s", m["Requests"], r.nl))
			b.WriteString(fmt.Sprintf("Data Downloaded: %.2f MB%s", m["BytesDownloaded"], r.nl))
			b.WriteString(fmt.Sprintf("Cache Hit Rate: %.2f%%%s", m["CacheHitRate"], r.nl))
			b.WriteString(fmt.Sprintf("4xx Error Rate: %.2f%%%s", m["4xxErrorRate"], r.nl))
			b.WriteString(fmt.Sprintf("5xx Error Rate: %.2f%%%s", m["5xxErrorRate"], r.nl))
			b.WriteString(fmt.Sprintf("Origin Latency: %.2f ms%s", m["OriginLatency"], r.nl))
			b.WriteString(r.nl)
		}
	}

	// DynamoDB
	if cfg.Services.DynamoDB.Enabled {
		if d, ok := allMetrics["dynamodb"]; ok {
			allTables := d.(map[string]any)
			for _, table := range cfg.Services.DynamoDB.TableNames {
				if td, ok := allTables[table]; ok {
					m := td.(map[string]float64)
					b.WriteString(fmt.Sprintf("%s %s%s", r.bold("DynamoDB"), r.esc(table), r.nl))
					b.WriteString(fmt.Sprintf("Total Requests: %.0f%s", m["RequestCount"], r.nl))
					b.WriteString(fmt.Sprintf("Read Throttles: %.0f%s", m["ReadThrottledRequests"], r.nl))
					b.WriteString(fmt.Sprintf("Write Throttles: %.0f%s", m["WriteThrottledRequests"], r.nl))
					b.WriteString(fmt.Sprintf("Latency: %.2f ms%s", m["SuccessfulRequestLatency"], r.nl))
					b.WriteString(fmt.Sprintf("Read Capacity: %.0f units%s", m["ConsumedReadCapacityUnits"], r.nl))
					b.WriteString(fmt.Sprintf("Write Capacity: %.0f units%s", m["ConsumedWriteCapacityUnits"], r.nl))
					totalErrors := m["UserErrors"] + m["SystemErrors"]
					b.WriteString(fmt.Sprintf("DB Errors: %.0f%s", totalErrors, r.nl))
					b.WriteString(r.nl)
				}
			}
		}
	}

	// RDS
	if cfg.Services.RDS.Enabled {
		if d, ok := allMetrics["rds"]; ok {
			m := d.(map[string]float64)

			var header string
			if cfg.Services.RDS.ClusterID != "" && cfg.Services.RDS.DBInstanceIdentifier != "" {
				header = fmt.Sprintf("%s %s / %s",
					r.bold("RDS"), r.esc(cfg.Services.RDS.ClusterID), r.esc(cfg.Services.RDS.DBInstanceIdentifier))
			} else if cfg.Services.RDS.ClusterID != "" {
				header = fmt.Sprintf("%s Cluster %s", r.bold("RDS"), r.esc(cfg.Services.RDS.ClusterID))
			} else {
				header = fmt.Sprintf("%s Instance %s", r.bold("RDS"), r.esc(cfg.Services.RDS.DBInstanceIdentifier))
			}
			b.WriteString(header + r.nl)

			if cfg.Services.RDS.DBInstanceIdentifier != "" {
				if v, ok := m["Instance_CPUUtilization_Average"]; ok {
					line := fmt.Sprintf("CPU: %.2f%% (avg)", v)
					if v2, ok2 := m["Instance_CPUUtilization_Maximum"]; ok2 {
						line += fmt.Sprintf(", %.2f%% (max)", v2)
					}
					b.WriteString(line + r.nl)
				}
				if v, ok := m["Instance_FreeableMemory"]; ok {
					b.WriteString(fmt.Sprintf("Free Memory: %.2f GB%s", v, r.nl))
				}
				if v, ok := m["Instance_DatabaseConnections"]; ok {
					b.WriteString(fmt.Sprintf("Connections: %.0f%s", v, r.nl))
				}
				if v, ok := m["Instance_ReadLatency"]; ok {
					b.WriteString(fmt.Sprintf("Read Latency: %.2f ms%s", v, r.nl))
				}
				if v, ok := m["Instance_WriteLatency"]; ok {
					b.WriteString(fmt.Sprintf("Write Latency: %.2f ms%s", v, r.nl))
				}
			}
			if cfg.Services.RDS.ClusterID != "" {
				if v, ok := m["Cluster_VolumeBytesUsed"]; ok {
					b.WriteString(fmt.Sprintf("Volume Size: %.2f GB%s", v, r.nl))
				}
				if v, ok := m["Cluster_VolumeReadIOPs"]; ok {
					b.WriteString(fmt.Sprintf("Read IOPS: %.0f%s", v, r.nl))
				}
				if v, ok := m["Cluster_VolumeWriteIOPs"]; ok {
					b.WriteString(fmt.Sprintf("Write IOPS: %.0f%s", v, r.nl))
				}
			}
			b.WriteString(r.nl)
		}
	}

	// WAF
	if cfg.Services.WAF.Enabled {
		if d, ok := allMetrics["waf"]; ok {
			m := d.(map[string]float64)
			b.WriteString(fmt.Sprintf("%s %s%s", r.bold("WAF"), r.esc(cfg.Services.WAF.WebACLName), r.nl))
			b.WriteString(fmt.Sprintf("Allowed Requests: %.0f%s", m["AllowedRequests"], r.nl))
			b.WriteString(fmt.Sprintf("Blocked Requests: %.0f%s", m["BlockedRequests"], r.nl))
			b.WriteString(r.nl)
		}
	}

	// CloudWatch Logs
	if cfg.Services.CloudWatchLogs.Enabled {
		if d, ok := allMetrics["cloudwatchLogs"]; ok {
			logsMetrics := d.(map[string]any)
			applicationLogs := make(map[string]any)
			lambdaLogs := make(map[string]any)

			for _, name := range cfg.Services.CloudWatchLogs.LogGroupNames {
				if data, ok := logsMetrics[name]; ok {
					if strings.Contains(name, "/aws/lambda/") {
						lambdaLogs[name] = data
					} else {
						applicationLogs[name] = data
					}
				}
			}

			if len(applicationLogs) > 0 {
				b.WriteString(r.bold("APPLICATION") + r.nl)
				for lg, data := range applicationLogs {
					cnt := data.(map[string]int)
					b.WriteString(fmt.Sprintf("%s:%s", r.esc(lg), r.nl))
					b.WriteString(fmt.Sprintf("INFO: %d%s", cnt["info"], r.nl))
					b.WriteString(fmt.Sprintf("WARN: %d%s", cnt["warn"], r.nl))
					b.WriteString(fmt.Sprintf("ERROR: %d%s", cnt["error"], r.nl))
					b.WriteString(r.nl)
				}
			}

			if len(lambdaLogs) > 0 {
				b.WriteString(r.bold("LAMBDA") + r.nl)
				for lg, data := range lambdaLogs {
					cnt := data.(map[string]int)
					b.WriteString(fmt.Sprintf("%s:%s", r.esc(lg), r.nl))
					b.WriteString(fmt.Sprintf("INFO: %d%s", cnt["info"], r.nl))
					b.WriteString(fmt.Sprintf("WARN: %d%s", cnt["warn"], r.nl))
					b.WriteString(fmt.Sprintf("ERROR: %d%s", cnt["error"], r.nl))
					b.WriteString(r.nl)
				}
			}
		}
	}

	// Footer
	b.WriteString(r.sep(timeParams.IsDailyReport))

	// Optionally wrap HTML
	if forEmail {
		return "<html><body style=\"font-family: monospace; white-space: pre-wrap;\">" + b.String() + "</body></html>"
	}
	return b.String()
}
