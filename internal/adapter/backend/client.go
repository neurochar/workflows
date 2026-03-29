package backend

import (
	privateCrmPb "github.com/neurochar/workflows/pkg/proto_pb/private/crm/v1"
	"google.golang.org/grpc"
)

type PrivateClientConnection = grpc.ClientConn

type Adapter interface {
	PrivateCC() *PrivateClientConnection
	PrivateCrmClient() privateCrmPb.CrmPrivateServiceClient
}
