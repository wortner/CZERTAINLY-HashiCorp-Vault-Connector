package discovery

import (
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/db"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/model"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/utils"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/vault"
	"context"
	"encoding/base64"
	vault2 "github.com/hashicorp/vault-client-go"
	"net/http"

	"go.uber.org/zap"
)

// DiscoveryAPIService is a service that implements the logic for the DiscoveryAPIServicer
// This service should implement the business logic for every endpoint for the DiscoveryAPI API.
// Include any external packages or services that will be required by this service.
type DiscoveryAPIService struct {
	discoveryRepo *db.DiscoveryRepository
	authorityRepo *db.AuthorityRepository
	log           *zap.Logger
}

// NewDiscoveryAPIService creates a default api service
func NewDiscoveryAPIService(discoveryRepo *db.DiscoveryRepository, authorityRepo *db.AuthorityRepository, logger *zap.Logger) DiscoveryAPIServicer {
	return &DiscoveryAPIService{
		discoveryRepo: discoveryRepo,
		authorityRepo: authorityRepo,
		log:           logger,
	}
}

// DeleteDiscovery - Delete Discovery
func (s *DiscoveryAPIService) DeleteDiscovery(ctx context.Context, uuid string) (model.ImplResponse, error) {
	discovery, err := s.discoveryRepo.FindDiscoveryByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{Message: "Discovery " + uuid + " not found."}), nil
	}
	err = s.discoveryRepo.DeleteDiscovery(discovery)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{Message: "Unable to delete discover" + discovery.UUID}), nil
	}

	return model.Response(204, nil), nil
}

// DiscoverCertificate - Initiate certificate Discovery
func (s *DiscoveryAPIService) DiscoverCertificate(ctx context.Context, discoveryRequestDto model.DiscoveryRequestDto) (model.ImplResponse, error) {
	response := model.DiscoveryProviderDto{
		Uuid:                        utils.DeterministicGUID(discoveryRequestDto.Name),
		Name:                        discoveryRequestDto.Name,
		Status:                      model.IN_PROGRESS,
		TotalCertificatesDiscovered: 0,
		CertificateData:             nil,
		Meta:                        nil,
	}
	discovery := &db.Discovery{
		UUID:         response.Uuid,
		Name:         response.Name,
		Status:       string(response.Status),
		Meta:         nil,
		Certificates: nil,
	}
	err := s.discoveryRepo.CreateDiscovery(discovery)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{Message: "Unable to create discovery " + discovery.UUID}), nil
	}
	uuid := model.GetAttributeFromArrayByUUID(model.DISCOVERY_AUTHORITY_ATTR, discoveryRequestDto.Attributes).GetContent()[0].GetData().(map[string]interface{})["uuid"].(string)
	enginesAttr := model.GetAttributeFromArrayByUUID(model.DISCOVERY_PKI_ENGINE_ATTR, discoveryRequestDto.Attributes)
	var enginesList []string
	if enginesAttr == nil {
		enginesList = nil

	} else {
		enginesList = make([]string, 0)
		for _, engine := range enginesAttr.GetContent() {
			engineData := engine.(model.ObjectAttributeContent).GetData().(map[string]interface{})
			engineName := engineData["engineName"].(string)
			enginesList = append(enginesList, engineName)
		}
	}

	authority, err := s.authorityRepo.FindAuthorityInstanceByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{Message: "Authority not found  " + uuid}), nil
	}
	go s.DiscoveryCertificates(authority, discovery, enginesList)

	return model.Response(http.StatusOK, response), nil
}

// GetDiscovery - Get Discovery status and result
func (s *DiscoveryAPIService) GetDiscovery(ctx context.Context, uuid string, discoveryDataRequestDto model.DiscoveryDataRequestDto) (model.ImplResponse, error) {
	discovery, err := s.discoveryRepo.FindDiscoveryByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{Message: "Discovery " + uuid + " not found."}), nil
	}
	if discovery.Status == "IN_PROGRESS" {
		return model.Response(http.StatusOK, model.DiscoveryProviderDto{Uuid: discovery.UUID, Name: discovery.Name, Status: model.IN_PROGRESS, TotalCertificatesDiscovered: 0, CertificateData: nil, Meta: nil}), nil
	} else {
		pagination := db.Pagination{
			Page:  int(discoveryDataRequestDto.PageNumber),
			Limit: int(discoveryDataRequestDto.ItemsPerPage),
		}
		result, _ := s.discoveryRepo.List(pagination)
		var certificateDtos []model.DiscoveryProviderCertificateDataDto
		rows, _ := result.Rows.([]*db.Certificate)
		for _, certificateData := range rows {
			discoveryProviderCertificateDataDto := model.DiscoveryProviderCertificateDataDto{
				Uuid:          certificateData.UUID,
				Base64Content: certificateData.Base64Content,
			}
			certificateDtos = append(certificateDtos, discoveryProviderCertificateDataDto)
		}

		return model.Response(http.StatusOK, model.DiscoveryProviderDto{Uuid: discovery.UUID, Name: discovery.Name, Status: model.COMPLETED, TotalCertificatesDiscovered: result.TotalRows, CertificateData: certificateDtos, Meta: nil}), nil
	}

}

func (s *DiscoveryAPIService) DiscoveryCertificates(authority *db.AuthorityInstance, discovery *db.Discovery, list []string) {
	// get the vault client
	client, err := vault.GetClient(*authority)
	if err != nil {
		discovery.Status = "FAILED"
		err := s.discoveryRepo.UpdateDiscovery(discovery)
		if err != nil {
			s.log.Error(err.Error())
		}
		s.log.Error(err.Error())
		return
	}
	// get the certificates
	ctx := context.Background()

	if list == nil {
		certificates, err := client.Secrets.PkiListCerts(ctx)
		if err != nil {
			discovery.Status = "FAILED"
			err := s.discoveryRepo.UpdateDiscovery(discovery)
			if err != nil {
				s.log.Error(err.Error())
			}
			return
		}
		var certificateKeys []*db.Certificate
		for _, certificateKey := range certificates.Data.Keys {
			certificateData, err := client.Secrets.PkiReadCert(ctx, certificateKey)
			if err != nil {
				discovery.Status = "FAILED"
				s.log.Error(err.Error())
				err := s.discoveryRepo.UpdateDiscovery(discovery)
				if err != nil {
					s.log.Error(err.Error())
				}

				return
			}
			certificate := db.Certificate{
				SerialNumber:  certificateKey,
				UUID:          utils.DeterministicGUID(certificateKey),
				Base64Content: base64.StdEncoding.EncodeToString([]byte(certificateData.Data.Certificate)),
			}
			certificateKeys = append(certificateKeys, &certificate)
		}
		err = s.discoveryRepo.AssociateCertificatesToDiscovery(discovery, certificateKeys...)
		if err != nil {
			discovery.Status = "FAILED"
			s.log.Error(err.Error())
			err := s.discoveryRepo.UpdateDiscovery(discovery)
			if err != nil {
				s.log.Error(err.Error())
			}
			return
		}
	} else {

		for _, engine := range list {
			certificates, err := client.Secrets.PkiListCerts(ctx, vault2.WithMountPath(engine))
			if err != nil {
				discovery.Status = "FAILED"
				err := s.discoveryRepo.UpdateDiscovery(discovery)
				if err != nil {
					s.log.Error(err.Error())
				}
				return
			}
			var certificateKeys []*db.Certificate
			for _, certificateKey := range certificates.Data.Keys {
				certificateData, err := client.Secrets.PkiReadCert(ctx, certificateKey)
				if err != nil {
					discovery.Status = "FAILED"
					s.log.Error(err.Error())
					err := s.discoveryRepo.UpdateDiscovery(discovery)
					if err != nil {
						s.log.Error(err.Error())
					}

					return
				}
				certificate := db.Certificate{
					SerialNumber:  certificateKey,
					UUID:          utils.DeterministicGUID(certificateKey),
					Base64Content: base64.StdEncoding.EncodeToString([]byte(certificateData.Data.Certificate)),
				}
				certificateKeys = append(certificateKeys, &certificate)
			}
			err = s.discoveryRepo.AssociateCertificatesToDiscovery(discovery, certificateKeys...)
			if err != nil {
				discovery.Status = "FAILED"
				s.log.Error(err.Error())
				err := s.discoveryRepo.UpdateDiscovery(discovery)
				if err != nil {
					s.log.Error(err.Error())
				}
				return
			}
		}
	}
	// Update discovery status to "COMPLETED"
	discovery.Status = "COMPLETED"
	err = s.discoveryRepo.UpdateDiscovery(discovery)
	if err != nil {
		discovery.Status = "FAILED"
		s.log.Error(err.Error())
		err := s.discoveryRepo.UpdateDiscovery(discovery)
		if err != nil {
			s.log.Error(err.Error())
		}
		return
	}

}
