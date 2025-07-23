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

type GlobalConfig struct {
	Notifications NotificationsConfig `json:"notifications"`
	Deployment    DeploymentConfig    `json:"deployment"`
	Monitoring    MonitoringConfig    `json:"monitoring"`
}

type NotificationsConfig struct {
	UseEmail bool           `json:"useEmail"`
	Email    EmailConfig    `json:"email"`
	Telegram TelegramConfig `json:"telegram"`
}

type EmailConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	HeaderFrom   string `json:"headerFrom"`
	EnvelopeFrom string `json:"envelopeFrom"`
	ToAddr       string `json:"toAddr"`
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
	DefaultPeriod      int `json:"defaultPeriod"`   // Hours
	DailyReportHourUTC int `json:"dailyReportHour"` // Hour of day (0-23)
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
	if config.Global.Notifications.UseEmail {
		if config.Global.Notifications.Email.Host == "" {
			return fmt.Errorf("email enabled but host is empty")
		}
		if config.Global.Notifications.Email.Port <= 0 {
			return fmt.Errorf("email enabled but port is invalid")
		}
		if config.Global.Notifications.Email.Username == "" || config.Global.Notifications.Email.Password == "" {
			return fmt.Errorf("email enabled but username/password missing")
		}
		if config.Global.Notifications.Email.HeaderFrom == "" {
			return fmt.Errorf("email enabled but headerFrom is empty")
		}
		if config.Global.Notifications.Email.EnvelopeFrom == "" {
			return fmt.Errorf("email enabled but envelopeFrom is empty")
		}
		if config.Global.Notifications.Email.ToAddr == "" {
			return fmt.Errorf("email enabled but toAddr is empty")
		}
	} else {
		// Telegram path
		if config.Global.Notifications.Telegram.BotToken == "" {
			return fmt.Errorf("telegram botToken is required when email is disabled")
		}
		if config.Global.Notifications.Telegram.ChatID == "" {
			return fmt.Errorf("telegram chatId is required when email is disabled")
		}
	}
	if config.Global.Deployment.LambdaFunctionName == "" {
		return fmt.Errorf("deployment lambdaFunctionName is required")
	}
	if config.Global.Monitoring.DailyReportHourUTC < 0 || config.Global.Monitoring.DailyReportHourUTC > 23 {
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
	nowUTC := time.Now().UTC()

	isDailyReport := nowUTC.Hour() == c.Global.Monitoring.DailyReportHourUTC

	var start time.Time
	if isDailyReport {
		start = nowUTC.Add(-24 * time.Hour)
	} else {
		start = nowUTC.Add(-time.Duration(c.Global.Monitoring.DefaultPeriod) * time.Hour)
	}

	return &TimeParams{
		StartTime:     start,
		EndTime:       nowUTC,
		IsDailyReport: isDailyReport,
		Location:      time.UTC,
	}, nil
}
