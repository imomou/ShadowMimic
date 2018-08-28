package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ecs"
)

//var service *SkynetCollectorDomain

// SkynetMetrics - temp model
type SkynetMetrics struct {
	ServiceName          string
	ServiceArn           string
	TaskDeinition        *ecs.TaskDefinition
	Dimension            []*cloudwatch.Dimension
	HealthyInstancesData []*cloudwatch.Datapoint
	CPUUtilisationData   []*cloudwatch.Datapoint
}

func main() {

	region := "ap-southeast-2"
	cluster := "Prod-EcsProductionCluster-EcsCluster-11106GYRQEVQV"
	elbNamespace := "AWS/ApplicationELB"
	eccNamespace := "AWS/ECS"
	healthyHostMetricName := "HealthyHostCount"
	cpuMetricName := "CPUUtilization"
	avgStats := "Average"

	startTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()

	sess := session.Must(session.NewSession(&aws.Config{
		Region: &region,
	}))

	svc := ecs.New(sess)
	cwl := cloudwatch.New(sess)

	i := 0
	token := ""
	serviceArns := make([]string, 0)

	for i < 10000 {

		req, error := svc.ListServices(&ecs.ListServicesInput{
			Cluster:   &cluster,
			NextToken: &token})

		if error != nil {
			fmt.Println(error)
		}

		for _, v := range req.ServiceArns {
			serviceArns = append(serviceArns, *v)
			i++
		}

		if req.NextToken == nil {
			break
		}
		token = *req.NextToken
	}

	token = ""
	i = 0
	servicelistMetrics := make([]cloudwatch.Metric, 0)

	for i < 10000 {

		serviceList, err := cwl.ListMetrics(&cloudwatch.ListMetricsInput{
			MetricName: &healthyHostMetricName,
			Namespace:  &elbNamespace})

		if err != nil {
			fmt.Println(err)
		}

		for _, v := range serviceList.Metrics {
			servicelistMetrics = append(servicelistMetrics, *v)
			i++
		}

		if serviceList.NextToken == nil {
			break
		}

		token = *serviceList.NextToken
	}

	sMetrics := make([]*SkynetMetrics, 0)

	for _, v := range serviceArns {

		bah, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Services: []*string{&v},
			Cluster:  &cluster})

		if err != nil {
			fmt.Println(err)
		}

		bah1, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: bah.Services[0].TaskDefinition})

		if err != nil {
			fmt.Println(err)
		}

		if bah.Services[0].LoadBalancers == nil || len(bah.Services[0].LoadBalancers) < 1 {
			continue
		}

		found := false
		for _, w := range servicelistMetrics {

			for _, z := range w.Dimensions {

				if strings.Compare(*z.Name, "TargetGroup") == 0 {

					if strings.Contains(*bah.Services[0].LoadBalancers[0].TargetGroupArn, *z.Value) {
						sMetrics = append(sMetrics, &SkynetMetrics{
							Dimension:     w.Dimensions,
							ServiceName:   *bah.Services[0].ServiceName,
							TaskDeinition: bah1.TaskDefinition,
							ServiceArn:    v})

						found = true
						break
					}
				}
			}

			if found == true {
				break
			}
		}
	}

	for _, a := range sMetrics {

		dimensions := make([]*cloudwatch.Dimension, 0)

		for _, dimension := range a.Dimension {

			if *dimension.Name == "TargetGroup" || *dimension.Name == "LoadBalancer" {
				dimensions = append(dimensions, dimension)
			}
		}

		hsMetrics, err := cwl.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			MetricName: &healthyHostMetricName,
			Namespace:  &elbNamespace,
			Dimensions: dimensions,
			Statistics: []*string{&avgStats},
			Period:     aws.Int64(60),
			StartTime:  &startTime,
			EndTime:    &endTime})

		if err != nil {
			fmt.Println(err)
		}

		cpuMetrics, err := cwl.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			MetricName: &cpuMetricName,
			Namespace:  &eccNamespace,
			Dimensions: []*cloudwatch.Dimension{&cloudwatch.Dimension{
				Name:  aws.String("ServiceName"),
				Value: &a.ServiceName},
				&cloudwatch.Dimension{
					Name:  aws.String("ClusterName"),
					Value: &cluster}},
			Statistics: []*string{&avgStats},
			Period:     aws.Int64(60),
			StartTime:  &startTime,
			EndTime:    &endTime})

		if err != nil {
			fmt.Println(err)
		}

		a.HealthyInstancesData = hsMetrics.Datapoints
		a.CPUUtilisationData = cpuMetrics.Datapoints
	}

	file, ferr := os.Create("details.csv")
	defer file.Close()

	if ferr != nil {
		fmt.Println(ferr)
		return
	}

	writer := csv.NewWriter(file)
	writer.Flush()
	writer.Write([]string{"StartTime", "ServiceName", "AllocatedCPU", "HealthyHosts", "CPUAverage"})

	for _, stuff := range sMetrics {

		asd := len(stuff.CPUUtilisationData)
		bfg := len(stuff.HealthyInstancesData)

		j := asd
		i := 0

		if asd < bfg {
			j = bfg
		}

		k := 0
		for i < j {

			var cpu float64
			if i < asd {
				cpu = *stuff.CPUUtilisationData[i].Average
			}

			//cpu = map[bool]float64{false: 0, true: *stuff.CPUUtilisationData[i].Average}[i < asd]
			cpuString := strconv.FormatFloat(cpu, 'f', 2, 64)
			var bah3 int64

			if i < len(stuff.CPUUtilisationData) || stuff.CPUUtilisationData == nil {
				bah3 = stuff.CPUUtilisationData[i].Timestamp.Unix()
			} else if stuff.HealthyInstancesData[i] != nil {
				bah3 = stuff.HealthyInstancesData[i].Timestamp.Unix()
			}

			timestamp := strconv.FormatInt(bah3, 10)

			var healthyHosts float64
			//healthyHosts := map[bool]float64{false: 0, true: *stuff.HealthyInstancesData[i].Average}[i < bfg]
			if i < bfg {
				healthyHosts = *stuff.HealthyInstancesData[i].Average
			}

			allocatedCPU := stuff.TaskDeinition.ContainerDefinitions[0].Cpu
			allocatedCPUStr := strconv.FormatInt(*allocatedCPU, 10)
			healthyHostsString := strconv.FormatFloat(healthyHosts, 'f', 0, 64)
			test := []string{timestamp, stuff.ServiceName, allocatedCPUStr, healthyHostsString, cpuString}

			asd := writer.Write(test)

			if asd != nil {
				fmt.Println(asd)
			}

			i++
			k++
		}
	}
	fmt.Println("done!")
}
