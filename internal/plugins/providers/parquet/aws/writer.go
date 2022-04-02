package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/natun-ai/natun/internal/plugin"
	"github.com/natun-ai/natun/internal/plugins/providers/parquet"
	"github.com/natun-ai/natun/pkg/api"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xitongsys/parquet-go-source/s3v2"
	"github.com/xitongsys/parquet-go/source"
	"time"
)

const pluginName = "parquet-aws"

func init() {
	plugin.Configurers.Register(pluginName, BindConfig)
	plugin.HistoricalWriterFactories.Register(pluginName, HistoricalWriterFactory)
}

func BindConfig(set *pflag.FlagSet) error {
	set.String("aws-access-key", "", "AWS Access Key - for historical data")
	set.String("aws-secret-key", "", "AWS Secret Key - for historical data")
	set.String("aws-region", "", "AWS Region - for historical data")
	set.String("s3-bucket", "", "S3 Bucket - for historical data")
	set.String("s3-basedir", "natun/features/", "S3 Base directory for storing features - for historical data")
	return nil
}

func HistoricalWriterFactory(viper *viper.Viper) (api.HistoricalWriter, error) {
	var opts []func(*config.LoadOptions) error
	if viper.GetString("aws-access-key") != "" && viper.GetString("aws-secret-key") != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     viper.GetString("aws-access-key"),
				SecretAccessKey: viper.GetString("aes-secret-key"),
			},
		}))
	}
	if viper.GetString("aws-region") != "" {
		opts = append(opts, config.WithRegion(viper.GetString("aws-region")))
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	bucket := viper.GetString("s3-bucket")
	if bucket == "" {
		return nil, fmt.Errorf("s3-bucket is required")
	}
	factory := sourceFactory(client, bucket, viper.GetString("s3-basedir"))

	return parquet.BaseParquet(4, factory), nil
}
func sourceFactory(client s3v2.S3API, bucket string, basedir string) parquet.SourceFactory {
	return func(ctx context.Context, fqn string) (source.ParquetFile, error) {
		d := time.Now().Format("2006-01-02")
		filename := fmt.Sprintf("%s/fqn=%s/timestamp=%s/data.snappy.parquet", basedir, fqn, d)
		return s3v2.NewS3FileWriterWithClient(ctx, client, bucket, filename, nil)
	}
}