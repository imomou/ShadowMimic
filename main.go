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
	"github.com/rs/xid"
)

type SkynetMetrics struct {
	ServiceName          string
	ServiceArn           string
	TaskDeinition        *ecs.TaskDefinition
	Dimension            []*cloudwatch.Dimension
	HealthyInstancesData []*cloudwatch.Datapoint
	CPUUtilisationData   []*cloudwatch.Datapoint
	RequestCount         []*cloudwatch.Datapoint
}

type ServiceDetail struct {
	ServiceName     string
	TempServiceName string
}

func main() {

	clusters := [...]string{"Prod-EcsBase-EcsCluster-1KWT57G8ND2EQ", "Prod-EcsDevCluster-EcsCluster-LI8U7BH3FTY2", "Prod-EcsProductionCluster-EcsCluster-11106GYRQEVQV", "Prod-EcsStagingCluster-EcsCluster-1AAGQRFIOE9AE", "Prod-WindowsDev-EcsCluster-1AIC5GZHFCJTI", "Prod-WindowsProduction-EcsCluster-N1TE86U9CWS2"}

	i := 1

	for i <= 14 {

		for _, cluster := range clusters {
			uglyFunction(&cluster, &i)
		}

		i++
	}

	fmt.Println("done!")
}

func uglyFunction(cluster *string, day *int) {

	region := "ap-southeast-2"
	elbNamespace := "AWS/ApplicationELB"
	eccNamespace := "AWS/ECS"
	healthyHostMetricName := "HealthyHostCount"
	cpuMetricName := "CPUUtilization"
	rcMetricName := "RequestCount"
	// lDimension := "LoadBalancer"
	avgStats := "Average"
	sumStats := "Sum"

	startTime := time.Now().AddDate(0, 0, (*day*-1)-1)
	endTime := time.Now().AddDate(0, 0, (*day * -1))

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
			Cluster:   cluster,
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
			Cluster:  cluster})

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

		//fmt.Println(dimensions)

		rcMetrics, err := cwl.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
			MetricName: &rcMetricName,
			Namespace:  &elbNamespace,
			Dimensions: dimensions,
			Statistics: []*string{&sumStats},
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
					Value: cluster}},
			Statistics: []*string{&avgStats},
			Period:     aws.Int64(60),
			StartTime:  &startTime,
			EndTime:    &endTime})

		if err != nil {
			fmt.Println(err)
		}

		a.HealthyInstancesData = hsMetrics.Datapoints
		a.CPUUtilisationData = cpuMetrics.Datapoints
		a.RequestCount = rcMetrics.Datapoints
	}

	fileName := *cluster + "-" + startTime.Format("20060102") + ".csv"

	file, ferr := os.Create(fileName)
	defer file.Close()

	if ferr != nil {
		fmt.Println(ferr)
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"TimeStampEpoch", "CusterName", "ServiceName", "AllocatedCPU", "HealthyHosts", "CPUAverage", "RequestCount"})

	servicesNames := make([]ServiceDetail, 0)

	var l int64
	for _, stuff := range sMetrics {

		asd := len(stuff.CPUUtilisationData)
		bfg := len(stuff.HealthyInstancesData)

		j := asd
		i := 0
		l++

		if asd < bfg {
			j = bfg
		}

		xid := xid.New()
		serviceName := "Service_" + xid.String()

		a := 0
		for a < len(servicesNames) {

			if servicesNames[a].ServiceName == stuff.ServiceName {
				serviceName = servicesNames[a].TempServiceName
				break
			}

			if a == len(servicesNames)-1 {
				servicesNames = append(servicesNames, ServiceDetail{ServiceName: stuff.ServiceName, TempServiceName: serviceName})
			}
		}

		k := 0
		for i < j-1 {

			var cpu float64
			if i < asd {
				cpu = *stuff.CPUUtilisationData[i].Average
			}

			cpuString := strconv.FormatFloat(cpu, 'f', 10, 64)
			var bah3 int64

			if i < len(stuff.CPUUtilisationData) && stuff.CPUUtilisationData != nil {
				bah3 = stuff.CPUUtilisationData[i].Timestamp.Unix()
			}

			timestampEpoch := strconv.FormatInt(bah3, 10)
			var healthyHosts float64
			if i < bfg {
				healthyHosts = *stuff.HealthyInstancesData[i].Average
			}

			var requestCount float64
			if i < len(stuff.RequestCount) {
				requestCount = *stuff.RequestCount[i].Sum
			}

			healthyHostsString := strconv.FormatFloat(healthyHosts, 'f', 0, 64)
			allocatedCPU := stuff.TaskDeinition.ContainerDefinitions[0].Cpu
			allocatedCPUStr := strconv.FormatInt(*allocatedCPU, 10)
			requestCountStr := strconv.FormatFloat(requestCount, 'f', 0, 64)
			test := []string{timestampEpoch, *cluster, serviceName, allocatedCPUStr, healthyHostsString, cpuString, requestCountStr}

			asd := writer.Write(test)

			if asd != nil {
				fmt.Println(asd)
			}

			i++
			k++
		}
	}

}
