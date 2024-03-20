package authority

import (
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/db"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/model"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/utils"
	"CZERTAINLY-HashiCorp-Vault-Connector/internal/vault"
	"context"
	"encoding/base64"
	"errors"
	vault2 "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"go.uber.org/zap"
	"net/http"
)

// CertificateManagementAPIService is a service that implements the logic for the CertificateManagementAPIServicer
// This service should implement the business logic for every endpoint for the CertificateManagementAPI API.
// Include any external packages or services that will be required by this service.
type CertificateManagementAPIService struct {
	authorityRepo *db.AuthorityRepository
	log           *zap.Logger
}

// NewCertificateManagementAPIService creates a default api service
func NewCertificateManagementAPIService(authorityRepo *db.AuthorityRepository, logger *zap.Logger) CertificateManagementAPIServicer {
	return &CertificateManagementAPIService{
		authorityRepo: authorityRepo,
		log:           logger,
	}
}

// IdentifyCertificate - Identify Certificate
func (s *CertificateManagementAPIService) IdentifyCertificate(ctx context.Context, uuid string, certificateIdentificationRequestDto model.CertificateIdentificationRequestDto) (model.ImplResponse, error) {
	raAttributes := certificateIdentificationRequestDto.RaProfileAttributes
	engineData := model.GetAttributeFromArrayByUUID(model.RA_PROFILE_ENGINE_ATTR, raAttributes).GetContent()[0].GetData().(map[string]interface{})
	engineName := engineData["engineName"].(string)
	authority, err := s.authorityRepo.FindAuthorityInstanceByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{
			Message: "Authority not found",
		}), nil
	}
	client, err := vault.GetClient(*authority)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil
	}
	serialNumber, err := utils.ExtractSerialNumber(certificateIdentificationRequestDto.Certificate)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}

	_, err = client.Secrets.PkiReadCert(ctx, serialNumber.Text(10), vault2.WithMountPath(engineName+"/"))
	if err != nil {
		s.log.Error(err.Error())
		return model.Response(http.StatusBadRequest, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	response := model.CertificateIdentificationResponseDto{
		Meta: nil,
	}

	return model.Response(http.StatusOK, response), errors.New("IdentifyCertificate method not implemented")
}

// IssueCertificate - Issue Certificate
func (s *CertificateManagementAPIService) IssueCertificate(ctx context.Context, uuid string, certificateSignRequestDto model.CertificateSignRequestDto) (model.ImplResponse, error) {
	raAttributes := certificateSignRequestDto.RaProfileAttributes
	engineData := model.GetAttributeFromArrayByUUID(model.RA_PROFILE_ENGINE_ATTR, raAttributes).GetContent()[0].GetData().(map[string]interface{})
	engineName := engineData["engineName"].(string)
	role := model.GetAttributeFromArrayByUUID(model.RA_PROFILE_ROLE_ATTR, raAttributes).GetContent()[0].GetData().(string)
	authority, err := s.authorityRepo.FindAuthorityInstanceByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{
			Message: "Authority not found",
		}), nil
	}
	client, err := vault.GetClient(*authority)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil
	}
	commonName, err := utils.ExtractCommonName(certificateSignRequestDto.Pkcs10)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	decoded, err := base64.StdEncoding.DecodeString(certificateSignRequestDto.Pkcs10)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil
	}

	signRequest := schema.PkiSignWithRoleRequest{
		CommonName: commonName,
		Csr:        string(decoded),
	}
	certificateSignResponse, err := client.Secrets.PkiSignWithRole(ctx, role, signRequest, vault2.WithMountPath(engineName+"/"))
	if err != nil {
		s.log.Error(err.Error())
		return model.Response(http.StatusBadRequest, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	certificate := certificateSignResponse.Data.Certificate
	serialNumber := certificateSignResponse.Data.SerialNumber

	CertificateDataResponseDto := model.CertificateDataResponseDto{
		CertificateData: base64.StdEncoding.EncodeToString([]byte(certificate)),
		Uuid:            utils.DeterministicGUID(serialNumber),
		Meta:            nil,
		CertificateType: "X.509",
	}

	return model.Response(http.StatusOK, CertificateDataResponseDto), nil
}

// ListIssueCertificateAttributes - List of Attributes to issue Certificate
func (s *CertificateManagementAPIService) ListIssueCertificateAttributes(ctx context.Context, uuid string) (model.ImplResponse, error) {
	return model.Response(http.StatusOK, nil), nil
}

// ListRevokeCertificateAttributes - List of Attributes to revoke Certificate
func (s *CertificateManagementAPIService) ListRevokeCertificateAttributes(ctx context.Context, uuid string) (model.ImplResponse, error) {
	return model.Response(http.StatusOK, nil), nil
}

// RenewCertificate - Renew Certificate
func (s *CertificateManagementAPIService) RenewCertificate(ctx context.Context, uuid string, certificateRenewRequestDto model.CertificateRenewRequestDto) (model.ImplResponse, error) {
	raAttributes := certificateRenewRequestDto.RaProfileAttributes
	engineData := model.GetAttributeFromArrayByUUID(model.RA_PROFILE_ENGINE_ATTR, raAttributes).GetContent()[0].GetData().(map[string]interface{})
	engineName := engineData["engineName"].(string)
	role := model.GetAttributeFromArrayByUUID(model.RA_PROFILE_ROLE_ATTR, raAttributes).GetContent()[0].GetData().(string)
	authority, err := s.authorityRepo.FindAuthorityInstanceByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{
			Message: "Authority not found",
		}), nil
	}

	client, err := vault.GetClient(*authority)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil
	}

	commonName, err := utils.ExtractCommonName(certificateRenewRequestDto.Pkcs10)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	signRequest := schema.PkiIssuerSignWithRoleRequest{
		CommonName: commonName,
		Csr:        certificateRenewRequestDto.Pkcs10,
	}
	certificateSignResponse, err := client.Secrets.PkiIssuerSignWithRole(ctx, "default", role, signRequest, vault2.WithMountPath(engineName+"/"))
	if err != nil {
		s.log.Error(err.Error())
		return model.Response(http.StatusBadRequest, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	certificate := certificateSignResponse.Data.Certificate
	serialNumber := certificateSignResponse.Data.SerialNumber

	CertificateDataResponseDto := model.CertificateDataResponseDto{
		CertificateData: base64.StdEncoding.EncodeToString([]byte(certificate)),
		Uuid:            utils.DeterministicGUID(serialNumber),
		Meta:            nil,
		CertificateType: "X.509",
	}

	return model.Response(http.StatusOK, CertificateDataResponseDto), nil
}

// RevokeCertificate - Revoke Certificate
func (s *CertificateManagementAPIService) RevokeCertificate(ctx context.Context, uuid string, certRevocationDto model.CertRevocationDto) (model.ImplResponse, error) {
	authority, err := s.authorityRepo.FindAuthorityInstanceByUUID(uuid)
	if err != nil {
		return model.Response(http.StatusNotFound, model.ErrorMessageDto{
			Message: "Authority not found",
		}), nil
	}
	client, err := vault.GetClient(*authority)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil
	}
	serialNumber, err := utils.ExtractSerialNumber(certRevocationDto.Certificate)
	if err != nil {
		return model.Response(http.StatusInternalServerError, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	revokeRequest := schema.PkiRevokeRequest{
		SerialNumber: serialNumber.Text(10),
	}
	_, err = client.Secrets.PkiRevoke(ctx, revokeRequest)
	if err != nil {
		s.log.Error(err.Error())
		return model.Response(http.StatusBadRequest, model.ErrorMessageDto{
			Message: err.Error(),
		}), nil

	}
	return model.Response(http.StatusOK, nil), nil

}

// ValidateIssueCertificateAttributes - Validate list of Attributes to issue Certificate
func (s *CertificateManagementAPIService) ValidateIssueCertificateAttributes(ctx context.Context, uuid string, requestAttributeDto []model.RequestAttributeDto) (model.ImplResponse, error) {
	return model.Response(http.StatusOK, nil), nil
}

// ValidateRevokeCertificateAttributes - Validate list of Attributes to revoke certificate
func (s *CertificateManagementAPIService) ValidateRevokeCertificateAttributes(ctx context.Context, uuid string, requestAttributeDto []model.RequestAttributeDto) (model.ImplResponse, error) {
	return model.Response(http.StatusOK, nil), nil
}
