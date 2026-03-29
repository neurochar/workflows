package workflows

import (
	"strings"
	"time"

	"github.com/neurochar/workflows/internal/infra/storage"
	"github.com/neurochar/workflows/internal/workflows/activity/backend_grpc_call"
	"github.com/neurochar/workflows/internal/workflows/activity/personal_data_remover"
	"github.com/neurochar/workflows/internal/workflows/activity/read_text_from_pdf"
	storageActivity "github.com/neurochar/workflows/internal/workflows/activity/storage"
	"github.com/neurochar/workflows/internal/workflows/activity/word2pdf"
	typesPb "github.com/neurochar/workflows/pkg/proto_pb/common/types"
	workflows_pb "github.com/neurochar/workflows/pkg/proto_pb/common/workflows"
	crmv1Pb "github.com/neurochar/workflows/pkg/proto_pb/private/crm/v1"
	"github.com/samber/lo"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type WorkflowProcessResumeFilePayload struct {
	TaskID     string
	Filename   string
	FileBucket string
	FileKey    string
}

type WorkflowProcessResumeFileResult struct {
	Status string
}

func (d *Controller) WorkflowProcessResumeFile(
	ctx workflow.Context,
	payload *workflows_pb.WorkflowProcessResumeFileInput,
) (resOutput *workflows_pb.WorkflowProcessResumeFileOutput, resError error) {
	defaultActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
		TaskQueue: workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_DEFAULT.String(),
	})

	storageActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
		TaskQueue: workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_STORAGE.String(),
	})

	word2pdfActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
		TaskQueue: workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_WORD2PDF.String(),
	})

	ocrActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    15 * time.Second,
			MaximumAttempts:    3,
		},
		TaskQueue: workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_OCR.String(),
	})

	pdRemoverActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
		TaskQueue: workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_PD_REMOVER.String(),
	})

	defer func() {
		if resError != nil {
			var res *backend_grpc_call.PatchCandidateResumeOutput
			_ = workflow.ExecuteActivity(
				defaultActivityCtx,
				ActivityPatchCandidateResume,
				&backend_grpc_call.PatchCandidateResumeInput{
					Req: &crmv1Pb.PatchCandidatesResumeRequest{
						Id:               payload.ResumeId,
						SkipVersionCheck: true,
						Payload: &crmv1Pb.PatchCandidatesResumeRequestPayload{
							Status: lo.ToPtr(typesPb.CandidateResumeStatus_CANDIDATE_RESUME_STATUS_PROCESS_ERROR),
							ErrorText: &crmv1Pb.PatchCandidatesResumeRequestPayload_ErrorText{
								Text: lo.ToPtr(resError.Error()),
							},
						},
					},
				},
			).Get(ctx, &res)
		}
	}()

	var readFileResult *storageActivity.GetFileOutput
	err := workflow.ExecuteActivity(
		storageActivityCtx,
		ActivityGetFile,
		&storageActivity.GetFileInput{
			Bucket: storage.BucketName(payload.Data.StorageBucket),
			Key:    payload.Data.StorageKey,
		},
	).Get(ctx, &readFileResult)
	if err != nil {
		return nil, err
	}

	fileData := readFileResult.Data

	if payload.Data.Type == workflows_pb.WorkflowProcessResumeFileInput_FILE_TYPE_WORD {
		var convertWordResult *word2pdf.WordToPDFOutput
		err = workflow.ExecuteActivity(
			word2pdfActivityCtx,
			ActivityConvertWordToPDF,
			&word2pdf.WordToPDFInput{
				Filename: payload.Data.Name,
				FileData: fileData,
			},
		).Get(ctx, &convertWordResult)
		if err != nil {
			return nil, err
		}

		fileData = convertWordResult.Data
	}

	var readTextResult *read_text_from_pdf.ReadTextFromPDFOutput
	err = workflow.ExecuteActivity(
		ocrActivityCtx,
		ActivityReadTextFromPDF,
		&read_text_from_pdf.ReadTextFromPDFInput{
			Filename: payload.Data.Name,
			FileData: fileData,
		},
	).Get(ctx, &readTextResult)
	if err != nil {
		return nil, err
	}

	text := strings.Join(readTextResult.Text, "\n\n")

	var removePDResult *personal_data_remover.AnonymizeOutput
	err = workflow.ExecuteActivity(
		pdRemoverActivityCtx,
		ActivityAnonymizeText,
		&personal_data_remover.AnonymizeInput{
			Text:     text,
			Language: "ru",
		},
	).Get(ctx, &removePDResult)
	if err != nil {
		return nil, err
	}

	text = removePDResult.AnonymizedText

	var patchResumeResult *backend_grpc_call.PatchCandidateResumeOutput
	err = workflow.ExecuteActivity(
		defaultActivityCtx,
		ActivityPatchCandidateResume,
		&backend_grpc_call.PatchCandidateResumeInput{
			Req: &crmv1Pb.PatchCandidatesResumeRequest{
				Id:               payload.ResumeId,
				SkipVersionCheck: true,
				Payload: &crmv1Pb.PatchCandidatesResumeRequestPayload{
					Status: lo.ToPtr(typesPb.CandidateResumeStatus_CANDIDATE_RESUME_STATUS_PROCESSED),
					AnalyzeData: &crmv1Pb.PatchCandidatesResumeRequestPayload_AnalyzeData{
						Data: &typesPb.CandidateResumeAnalyzeData{
							AnonymizedText: text,
							DataVersion:    1,
						},
					},
					ErrorText: &crmv1Pb.PatchCandidatesResumeRequestPayload_ErrorText{},
				},
			},
		},
	).Get(ctx, &patchResumeResult)
	if err != nil {
		return nil, err
	}

	return &workflows_pb.WorkflowProcessResumeFileOutput{
		Status: "OK",
	}, nil
}
