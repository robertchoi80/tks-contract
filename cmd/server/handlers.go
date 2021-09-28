package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openinfradev/tks-contract/pkg/contract"
	gc "github.com/openinfradev/tks-contract/pkg/grpc-client"
	"github.com/openinfradev/tks-contract/pkg/log"
	pb "github.com/openinfradev/tks-proto/pbgo"
	"github.com/openinfradev/tks-cluster-lcm/pkg/argowf"
)

var (
	argowfClient *argowf.Client
	contractAccessor *contract.Accessor
	cspInfoClient    *gc.CspInfoServiceClient
)

func InitHandlers( argoAddress string, argoPort int ) {
	_client, err := argowf.New( argoAddress, argoPort, false, "" );
	if err != nil {
		log.Fatal( "failed to create argowf client : ", err )
	}
	argowfClient = _client;
}

// CreateContract implements pbgo.ContractService.CreateContract gRPC
func (s *server) CreateContract(ctx context.Context, in *pb.CreateContractRequest) (*pb.CreateContractResponse, error) {
	log.Info("Request 'CreateContract' for contract name", in.GetContractorName())
	contractId, err := contractAccessor.Create(in.GetContractorName(), in.GetAvailableServices(), in.GetQuota())
	if err != nil {
		res := pb.CreateContractResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}
		return &res, err
	}
	log.Info("newly created Contract Id:", contractId)

	res, err := cspInfoClient.CreateCSPInfo(ctx, contractId.String(), in.GetCspName(), in.GetCspAuth())
	log.Info("newly created CSP Id:", res.GetId())
	if err != nil || res.GetCode() != pb.Code_OK_UNSPECIFIED {
		res := pb.CreateContractResponse{
			Code: res.GetCode(),
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}
		return &res, err
	}

	// check workflow
	{
		nameSpace := "argo"
		if err := argowfClient.IsRunningWorkflowByContractId(nameSpace, contractId.String()); err != nil {
			log.Error(fmt.Sprintf("Already running workflow. contractId : %s", contractId.String()))
			return &pb.CreateContractResponse{
				Code: pb.Code_OK_UNSPECIFIED,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Already running workflow. contractId : %s", contractId.String() ),
				},
			}, nil
		}
	}

	// call workflow
	{
		workflow := "tks-create-contract-repo"
		nameSpace := "argo"

		opts := argowf.SubmitOptions{}
		opts.Parameters = []string{ 
			"contract_id=" + contractId.String(), 
		};

		res, err := argowfClient.SumbitWorkflowFromWftpl( workflow, nameSpace, opts );
		if err != nil {
			log.Error( "failed to submit argo workflow %s template. err : %s", workflow, err )
			return &pb.CreateContractResponse {
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
				},
			}, nil
		}
		log.Debug("submited workflow template :", res)
	}

	// [TODO] Contract status 관리?

	return &pb.CreateContractResponse{
		Code:       pb.Code_OK_UNSPECIFIED,
		Error:      nil,
		CspId:      res.GetId(),
		ContractId: contractId.String(),
	}, nil
}

// UpdateQuota implements pbgo.ContractService.UpdateQuota gRPC
func (s *server) UpdateQuota(ctx context.Context, in *pb.UpdateQuotaRequest) (*pb.UpdateQuotaResponse, error) {
	log.Info("Request 'UpdateQuota' for contract id ", in.GetContractId())
	contractID, err := uuid.Parse(in.GetContractId())
	if err != nil {
		res := pb.UpdateQuotaResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("invalid contract ID %s", in.GetContractId()),
			},
		}
		return &res, err
	}
	prev, curr, err := contractAccessor.UpdateResourceQuota(contractID, in.GetQuota())

	if err != nil {
		res := pb.UpdateQuotaResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}
		return &res, err
	}
	return &pb.UpdateQuotaResponse{
		Code:         pb.Code_OK_UNSPECIFIED,
		Error:        nil,
		PrevQuota:    prev,
		CurrentQuota: curr,
	}, nil
}

// UpdateServices implements pbgo.ContractService.UpdateServices gRPC
func (s *server) UpdateServices(ctx context.Context, in *pb.UpdateServicesRequest) (*pb.UpdateServicesResponse, error) {
	log.Info("Request 'UpdateServices' for contract id ", in.GetContractId())
	contractID, err := uuid.Parse(in.GetContractId())
	if err != nil {
		res := pb.UpdateServicesResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("invalid contract ID %s", in.GetContractId()),
			},
		}
		return &res, err
	}
	prev, curr, err := contractAccessor.UpdateAvailableServices(contractID, in.GetAvailableServices())
	if err != nil {
		res := pb.UpdateServicesResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}
		return &res, err
	}
	return &pb.UpdateServicesResponse{
		Code:            pb.Code_OK_UNSPECIFIED,
		Error:           nil,
		PrevServices:    prev,
		CurrentServices: curr,
	}, nil
}

// GetContract implements pbgo.ContractService.GetContract gRPC
func (s *server) GetContract(ctx context.Context, in *pb.GetContractRequest) (*pb.GetContractResponse, error) {
	log.Info("Request 'GetContract' for contract id ", in.GetContractId())
	contractID, err := uuid.Parse(in.GetContractId())
	if err != nil {
		res := pb.GetContractResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("invalid contract ID %s", in.GetContractId()),
			},
		}
		return &res, err
	}
	contract, err := contractAccessor.GetContract(contractID)
	if err != nil {
		res := pb.GetContractResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}
		return &res, err
	}
	res := pb.GetContractResponse{
		Code:     pb.Code_OK_UNSPECIFIED,
		Error:    nil,
		Contract: contract,
	}
	return &res, nil
}

// GetQuota implements pbgo.ContractService.GetContract gRPC
func (s *server) GetQuota(ctx context.Context, in *pb.GetQuotaRequest) (*pb.GetQuotaResponse, error) {
	log.Info("Request 'GetQuota' for contract id ", in.GetContractId())
	contractID, err := uuid.Parse(in.GetContractId())
	if err != nil {
		return &pb.GetQuotaResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("invalid contract ID %s", in.GetContractId()),
			},
		}, err
	}

	quota, err := contractAccessor.GetResourceQuota(contractID)
	if err != nil {
		return &pb.GetQuotaResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: err.Error(),
			},
		}, err
	}
	return &pb.GetQuotaResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Quota: &pb.ContractQuota{
			Cpu:      quota.Cpu,
			Memory:   quota.Memory,
			Block:    quota.Block,
			BlockSsd: quota.BlockSsd,
			Fs:       quota.Fs,
			FsSsd:    quota.FsSsd,
		},
	}, nil
}

// GetAvailableServices implements pbgo.ContractService.GetAvailableServices gRPC
func (s *server) GetAvailableServices(ctx context.Context, in *pb.GetAvailableServicesRequest) (*pb.GetAvailableServicesResponse, error) {
	log.Info("Request 'GetAvailableServices' for contract id ", in.GetContractId())
	contractID, err := uuid.Parse(in.GetContractId())
	if err != nil {
		return &pb.GetAvailableServicesResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("invalid contract ID %s", in.GetContractId()),
			},
		}, err
	}

	contract, err := contractAccessor.GetContract(contractID)
	if err != nil {
		return nil, fmt.Errorf("could not find contract for contract id %s", contractID)
	}
	res := pb.GetAvailableServicesResponse{
		Code:                pb.Code_OK_UNSPECIFIED,
		Error:               nil,
		AvaiableServiceApps: contract.AvailableServices,
	}
	return &res, nil
}
