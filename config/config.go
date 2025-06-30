package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed config.json
var configData []byte

func LoadEmbeddedConfig() (*Config, error) {
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing embedded config JSON: %v", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("embedded config validation failed: %v", err)
	}

	return &config, nil
}

type TelegramConfig struct {
	BotToken string `json:"botToken"`
	ChatID   string `json:"chatId"`
}

type DeploymentConfig struct {
	LambdaFunctionName   string `json:"lambdaFunctionName"`
	LambdaCronExpression string `json:"lambdaCronExpression"`
}

type MonitoringConfig struct {
	Timezone        string `json:"timezone"`
	DefaultPeriod   int    `json:"defaultPeriod"`   // Hours
	DailyReportHour int    `json:"dailyReportHour"` // Hour of day (0-23)
}

type GlobalConfig struct {
	Telegram   TelegramConfig   `json:"telegram"`
	Deployment DeploymentConfig `json:"deployment"`
	Monitoring MonitoringConfig `json:"monitoring"`
}

type ServiceConfig struct {
	EC2 struct {
		Enabled    bool   `json:"enabled"`
		InstanceID string `json:"instanceId"`
	} `json:"ec2"`

	S3 struct {
		Enabled    bool   `json:"enabled"`
		BucketName string `json:"bucketName"`
	} `json:"s3"`

	ALB struct {
		Enabled bool   `json:"enabled"`
		ALBName string `json:"albName"`
	} `json:"alb"`

	CloudFront struct {
		Enabled        bool   `json:"enabled"`
		DistributionID string `json:"distributionId"`
	} `json:"cloudfront"`

	CloudWatchAgent struct {
		Enabled    bool   `json:"enabled"`
		InstanceID string `json:"instanceId"`
	} `json:"cloudwatchAgent"`

	CloudWatchLogs struct {
		Enabled       bool     `json:"enabled"`
		LogGroupNames []string `json:"logGroupNames"`
	} `json:"cloudwatchLogs"`

	WAF struct {
		Enabled    bool   `json:"enabled"`
		WebACLID   string `json:"webACLId"`
		WebACLName string `json:"webACLName"`
	} `json:"waf"`

	DynamoDB struct {
		Enabled    bool     `json:"enabled"`
		TableNames []string `json:"tableNames"`
	} `json:"dynamodb"`

	RDS struct {
		Enabled              bool   `json:"enabled"`
		ClusterID            string `json:"clusterId"`
		DBInstanceIdentifier string `json:"dbInstanceIdentifier"`
	} `json:"rds"`
}

type Config struct {
	Global   GlobalConfig  `json:"global"`
	Services ServiceConfig `json:"services"`
}

func validateConfig(config *Config) error {
	if config.Global.Telegram.BotToken == "" {
		return fmt.Errorf("telegram botToken is required")
	}
	if config.Global.Telegram.ChatID == "" {
		return fmt.Errorf("telegram chatId is required")
	}
	if config.Global.Deployment.LambdaFunctionName == "" {
		return fmt.Errorf("deployment lambdaFunctionName is required")
	}
	if config.Global.Monitoring.Timezone == "" {
		return fmt.Errorf("monitoring timezone is required")
	}
	if _, err := time.LoadLocation(config.Global.Monitoring.Timezone); err != nil {
		return fmt.Errorf("invalid timezone '%s': %v", config.Global.Monitoring.Timezone, err)
	}
	if config.Global.Monitoring.DailyReportHour < 0 || config.Global.Monitoring.DailyReportHour > 23 {
		return fmt.Errorf("dailyReportHour must be between 0 and 23")
	}
	if config.Global.Monitoring.DefaultPeriod <= 0 {
		return fmt.Errorf("defaultPeriod must be greater than 0")
	}

	if config.Services.EC2.Enabled && config.Services.EC2.InstanceID == "" {
		return fmt.Errorf("EC2 is enabled but instanceId is empty")
	}
	if config.Services.S3.Enabled && config.Services.S3.BucketName == "" {
		return fmt.Errorf("S3 is enabled but bucketName is empty")
	}
	if config.Services.ALB.Enabled && config.Services.ALB.ALBName == "" {
		return fmt.Errorf("ALB is enabled but albName is empty")
	}
	if config.Services.CloudFront.Enabled && config.Services.CloudFront.DistributionID == "" {
		return fmt.Errorf("CloudFront is enabled but distributionId is empty")
	}
	if config.Services.CloudWatchAgent.Enabled && config.Services.CloudWatchAgent.InstanceID == "" {
		return fmt.Errorf("CloudWatch Agent is enabled but instanceId is empty")
	}
	if config.Services.CloudWatchLogs.Enabled && len(config.Services.CloudWatchLogs.LogGroupNames) == 0 {
		return fmt.Errorf("CloudWatch Logs is enabled but logGroupNames array is empty")
	}
	if config.Services.WAF.Enabled {
		if config.Services.WAF.WebACLID == "" {
			return fmt.Errorf("WAF is enabled but webACLId is empty")
		}
		if config.Services.WAF.WebACLName == "" {
			return fmt.Errorf("WAF is enabled but webACLName is empty")
		}
	}
	if config.Services.DynamoDB.Enabled && len(config.Services.DynamoDB.TableNames) == 0 {
		return fmt.Errorf("DynamoDB is enabled but tableNames array is empty")
	}
	if config.Services.RDS.Enabled {
		if config.Services.RDS.ClusterID == "" && config.Services.RDS.DBInstanceIdentifier == "" {
			return fmt.Errorf("RDS is enabled but both clusterId and dbInstanceIdentifier are empty - at least one is required")
		}
	}

	return nil
}

type TimeParams struct {
	StartTime     time.Time
	EndTime       time.Time
	IsDailyReport bool
	Location      *time.Location
}

func (c *Config) GetTimeParams() (*TimeParams, error) {
	loc, err := time.LoadLocation(c.Global.Monitoring.Timezone)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(loc)
	isDailyReport := now.Hour() == c.Global.Monitoring.DailyReportHour

	var startTime time.Time
	if isDailyReport {
		// Daily report: look back 24 hours
		startTime = now.Add(-24 * time.Hour)
	} else {
		// Regular report: use configured period
		startTime = now.Add(-time.Duration(c.Global.Monitoring.DefaultPeriod) * time.Hour)
	}

	return &TimeParams{
		StartTime:     startTime,
		EndTime:       now,
		IsDailyReport: isDailyReport,
		Location:      loc,
	}, nil
}
