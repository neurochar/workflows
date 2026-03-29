package backend_grpc_call

import (
	"context"
	"log/slog"

	backendAdapter "github.com/neurochar/workflows/internal/adapter/backend"
	crmv1 "github.com/neurochar/workflows/pkg/proto_pb/private/crm/v1"
	"go.temporal.io/sdk/activity"
)

type Activity struct {
	adapter backendAdapter.Adapter
	logger  *slog.Logger
}

func New(adapter backendAdapter.Adapter, logger *slog.Logger) *Activity {
	return &Activity{
		adapter: adapter,
		logger:  logger,
	}
}

type PatchCandidateResumeInput struct {
	Req *crmv1.PatchCandidatesResumeRequest
}

type PatchCandidateResumeOutput struct {
	Result *crmv1.PatchCandidatesResumeResponse
}

func (d *Activity) PatchCandidateResume(ctx context.Context, payload *PatchCandidateResumeInput) (*PatchCandidateResumeOutput, error) {
	activity.RecordHeartbeat(ctx, "start")

	d.logger.Info("patching candidate resume",
		slog.String("candidate_id", payload.Req.Id),
	)

	activity.RecordHeartbeat(ctx, "sending_grpc_request")

	result, err := d.adapter.PrivateCrmClient().PatchCandidatesResume(ctx, payload.Req)
	if err != nil {
		d.logger.Error("failed to patch candidate resume",
			slog.String("candidate_id", payload.Req.Id),
			slog.Any("err", err),
		)
		return nil, err
	}

	activity.RecordHeartbeat(ctx, "done")

	return &PatchCandidateResumeOutput{
		Result: result,
	}, nil
}
