package services

import (
	"errors"
	"github.com/jinzhu/gorm"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm/converters"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError)
	GetClusterDNS(clusterID string) (string, *ocmErrors.ServiceError)
	ListByStatus(state api.ClusterStatus) ([]api.Cluster, *ocmErrors.ServiceError)
	UpdateStatus(id string, status api.ClusterStatus) error
	FindCluster(criteria FindClusterCriteria) (*api.Cluster, *ocmErrors.ServiceError)
}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	ocmClient         ocm.Client
	awsConfig         *config.AWSConfig
	clusterBuilder    ocm.ClusterBuilder
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, ocmClient ocm.Client, awsConfig *config.AWSConfig) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		ocmClient:         ocmClient,
		awsConfig:         awsConfig,
		clusterBuilder:    ocm.NewClusterBuilder(awsConfig),
	}
}

// Create creates a new OSD cluster
//
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	// Build a new OSD cluster object
	newCluster, err := c.clusterBuilder.NewOCMClusterFromCluster(cluster)
	if err != nil {
		return nil, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := c.ocmClient.CreateCluster(newCluster)
	if err != nil {
		return createdCluster, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	// convert the cluster to the cluster type this service understands before saving
	if err := dbConn.Save(converters.ConvertCluster(createdCluster)).Error; err != nil {
		return &clustersmgmtv1.Cluster{}, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	return createdCluster, nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *ocmErrors.ServiceError) {
	ingresses, err := c.ocmClient.GetClusterIngresses(clusterID)
	if err != nil {
		return "", ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	var clusterDNS string
	ingresses.Items().Each(func(ingress *clustersmgmtv1.Ingress) bool {
		if ingress.Default() == true {
			clusterDNS = ingress.DNSName()
			return false
		}
		return true
	})

	return clusterDNS, nil
}

func (c clusterService) ListByStatus(status api.ClusterStatus) ([]api.Cluster, *ocmErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var clusters []api.Cluster

	if err := dbConn.Model(&api.Cluster{}).Where("status = ?", status).Scan(&clusters).Error; err != nil {
		return nil, ocmErrors.GeneralError("failed to query by status: %s", err.Error())
	}

	return clusters, nil
}

func (c clusterService) UpdateStatus(id string, status api.ClusterStatus) error {
	dbConn := c.connectionFactory.New()

	if err := dbConn.Table("clusters").Where("id = ?", id).Update("status", status).Error; err != nil {
		return ocmErrors.GeneralError("failed to update status: %s", err.Error())
	}

	return nil
}

type FindClusterCriteria struct {
	Provider string
	Region   string
	MultiAZ  bool
	Status   api.ClusterStatus
}

func (c clusterService) FindCluster(criteria FindClusterCriteria) (*api.Cluster, *ocmErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
	}

	if err := dbConn.Where(clusterDetails).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, ocmErrors.GeneralError("failed to find cluster with criteria: %s", err.Error())
	}

	return &cluster, nil
}