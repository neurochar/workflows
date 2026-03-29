// Package config contains app config struct
package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config struct for app config
type Config struct {
	WorkflowsApp struct {
		Name    string `yaml:"name" env:"WORKFLOWS_APP_NAME" env-default:"backend"`
		Version string `yaml:"version" env:"WORKFLOWS_APP_VERSION" env-default:"1.0.0"`
		Base    struct {
			StartTimeoutSec int  `yaml:"start_timeout_sec" env:"WORKFLOWS_APP_BASE_START_TIMEOUT_SEC" env-default:"10"`
			StopTimeoutSec  int  `yaml:"stop_timeout_sec" env:"WORKFLOWS_APP_BASE_STOP_TIMEOUT_SEC" env-default:"2"`
			IsProd          bool `yaml:"is_prod" env:"WORKFLOWS_APP_BASE_IS_PROD" env-default:"false"`
			UseFxLogger     bool `yaml:"use_fx_logger" env:"WORKFLOWS_APP_BASE_USE_FX_LOGGER" env-default:"true"`
			UseLogger       bool `yaml:"use_logger" env:"WORKFLOWS_APP_BASE_USE_LOGGER" env-default:"true"`
		} `yaml:"base"`
	} `yaml:"workflows_app"`
	Temporal struct {
		Host      string `yaml:"host" env:"TEMPORAL_HOST" env-default:"127.0.0.1:7233"`
		Namespace string `yaml:"namespace" env:"TEMPORAL_NAMESPACE" env-default:"default"`
	} `yaml:"temporal"`
	Storage struct {
		S3Endpoint       string `yaml:"s3_endpoint" env:"STORAGE_S3_ENDPOINT" env-default:""`
		S3AccessKey      string `yaml:"s3_access_key" env:"STORAGE_S3_ACCESS_KEY" env-default:""`
		S3SecretKey      string `yaml:"s3_secret_key" env:"STORAGE_S3_SECRET_KEY" env-default:""`
		S3Region         string `yaml:"s3_region" env:"STORAGE_S3_REGION" env-default:""`
		S3URL            string `yaml:"s3_url" env:"STORAGE_S3_URL" env-default:""`
		S3URLIsHost      bool   `yaml:"s3_url_is_host" env:"STORAGE_S3_URL_IS_HOST" env-default:"false"`
		S3URLHostPrefix  string `yaml:"s3_url_host_prefix" env:"STORAGE_S3_URL_HOST_PREFIX" env-default:""`
		S3URLHostPostfix string `yaml:"s3_url_host_postfix" env:"STORAGE_S3_URL_HOST_POSTFIX" env-default:""`
	} `yaml:"storage"`
	Backend struct {
		GRPCPrivateEndpoint string `yaml:"grpc_private_endpoint" env:"BACKEND_GRPC_PRIVATE_ENDPOINT" env-default:""`
	} `yaml:"backend"`
	Workers struct {
		Ocr struct {
			Service         string `yaml:"service" env:"WORKERS_OCR_SERVICE" env-default:""`
			Readiness       string `yaml:"readiness" env:"WORKERS_OCR_READINESS" env-default:""`
			FallbackService string `yaml:"fallback_service" env:"WORKERS_OCR_FALLBACK_SERVICE" env-default:""`
		} `yaml:"ocr"`
		Word2pdf struct {
			Service   string `yaml:"service" env:"WORKERS_WORD2PDF_SERVICE" env-default:""`
			Readiness string `yaml:"readiness" env:"WORKERS_WORD2PDF_READINESS" env-default:""`
		} `yaml:"word2pdf"`
		PDRemover struct {
			Service   string `yaml:"service" env:"WORKERS_PD_REMOVER_SERVICE" env-default:""`
			Readiness string `yaml:"readiness" env:"WORKERS_PD_REMOVER_READINESS" env-default:""`
		} `yaml:"pd_remover"`
	} `yaml:"workers"`
}

// LoadConfig loads app config from file
func LoadConfig(files ...string) Config {
	var Config Config

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			err := cleanenv.ReadConfig(file, &Config)
			if err != nil {
				log.Println("config file error", err)
			}
		} else {
			log.Println("config file not found", file)
		}
	}

	return Config
}
