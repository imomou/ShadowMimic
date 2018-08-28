package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

//SkynetRepository - interface for skynet events repo
type SkynetRepository interface {
	ListMetrics(metric *string, namespace *string) cloudwatch.ListMetricsOutput
}

//SkynetCloudWatchRepository - implementation
type SkynetCloudWatchRepository struct {
	service *cloudwatch.CloudWatch
}

func newSkynetCloudWatchRepository(cwlService *cloudwatch.CloudWatch) *SkynetCloudWatchRepository {
	return &SkynetCloudWatchRepository{service: cwlService}
}

//ListrMetrics - Return list of metrics
func (cwlRepo SkynetCloudWatchRepository) ListrMetrics(metric *string, namespace *string) *cloudwatch.ListMetricsOutput {

	input := cloudwatch.ListMetricsInput{
		MetricName: metric,
		Namespace:  namespace}

	results, err := cwlRepo.service.ListMetrics(&input)

	if err != nil {
		fmt.Println(err)
	}

	return results
}
