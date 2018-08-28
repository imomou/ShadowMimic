package main

//SkynetDomain - Get statistics for each of the metrics
type SkynetDomain interface {
	GetMetrics()
}

//SkynetCollectorDomain -
type SkynetCollectorDomain struct {
	repository *SkynetCloudWatchRepository
}

func newSkynetCollectorDomain(repo *SkynetCloudWatchRepository) *SkynetCollectorDomain {
	return &SkynetCollectorDomain{repository: repo}
}

//GetMetrics -
func (domain SkynetCollectorDomain) GetMetrics() {

}
