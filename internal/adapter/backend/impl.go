package backend

import (
	"log/slog"

	privateCrmPb "github.com/neurochar/workflows/pkg/proto_pb/private/crm/v1"
)

type AdapterImpl struct {
	privateCrmClient privateCrmPb.CrmPrivateServiceClient
	privateCC        *PrivateClientConnection
	logger           *slog.Logger
}

func NewAdapterImpl(
	privateCC *PrivateClientConnection,
	logger *slog.Logger,
) (*AdapterImpl, error) {
	const op = "NewAdapterImpl"

	adapter := &AdapterImpl{
		privateCrmClient: privateCrmPb.NewCrmPrivateServiceClient(privateCC),
		logger:           logger,
		privateCC:        privateCC,
	}

	return adapter, nil
}

func (a *AdapterImpl) PrivateCC() *PrivateClientConnection {
	return a.privateCC
}

func (a *AdapterImpl) PrivateCrmClient() privateCrmPb.CrmPrivateServiceClient {
	return a.privateCrmClient
}
