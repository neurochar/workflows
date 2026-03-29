package workflows

import (
	"log/slog"

	temporalWorker "github.com/neurochar/workflows/internal/infra/temporal/worker"
	"github.com/neurochar/workflows/internal/workflows/activity/backend_grpc_call"
	"github.com/neurochar/workflows/internal/workflows/activity/personal_data_remover"
	"github.com/neurochar/workflows/internal/workflows/activity/read_text_from_pdf"
	"github.com/neurochar/workflows/internal/workflows/activity/storage"
	"github.com/neurochar/workflows/internal/workflows/activity/word2pdf"
	workflows_pb "github.com/neurochar/workflows/pkg/proto_pb/common/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	ActivityReadTextFromPDF      = "ReadTextFromPDF"
	ActivityConvertWordToPDF     = "ConvertWordToPDF"
	ActivityAnonymizeText        = "AnonymizeText"
	ActivityGetFile              = "GetFile"
	ActivityPatchCandidateResume = "PatchCandidateResume"
)

type Controller struct {
	logger          *slog.Logger
	workerClient    temporalWorker.WorkerClient
	defaultWorker   worker.Worker
	ocrWorker       worker.Worker
	storageWorker   worker.Worker
	word2pdfWorker  worker.Worker
	pdRemoverWorker worker.Worker

	activityReadTextFromPdf  *read_text_from_pdf.Activity
	activityConvertWordToPDF *word2pdf.Activity
	activityPDRemover        *personal_data_remover.Activity
	activityStorage          *storage.Activity
	activityBackendGrpcCall  *backend_grpc_call.Activity
}

func NewController(
	logger *slog.Logger,
	workerClient temporalWorker.WorkerClient,
	activityReadTextFromPdf *read_text_from_pdf.Activity,
	activityConvertWordToPDF *word2pdf.Activity,
	activityPDRemover *personal_data_remover.Activity,
	activityStorage *storage.Activity,
	activityBackendGrpcCall *backend_grpc_call.Activity,
) *Controller {
	return &Controller{
		logger:                   logger,
		workerClient:             workerClient,
		activityReadTextFromPdf:  activityReadTextFromPdf,
		activityConvertWordToPDF: activityConvertWordToPDF,
		activityPDRemover:        activityPDRemover,
		activityStorage:          activityStorage,
		activityBackendGrpcCall:  activityBackendGrpcCall,
	}
}

func (ctrl *Controller) RegisterWorkers() {
	ctrl.defaultWorker = worker.New(
		ctrl.workerClient,
		workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_DEFAULT.String(),
		worker.Options{
			MaxConcurrentActivityExecutionSize:     10,
			MaxConcurrentWorkflowTaskExecutionSize: 10,
		},
	)

	ctrl.ocrWorker = worker.New(
		ctrl.workerClient,
		workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_OCR.String(),
		worker.Options{
			MaxConcurrentActivityExecutionSize: 2,
		},
	)

	ctrl.storageWorker = worker.New(
		ctrl.workerClient,
		workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_STORAGE.String(),
		worker.Options{
			MaxConcurrentActivityExecutionSize: 3,
		},
	)

	ctrl.word2pdfWorker = worker.New(
		ctrl.workerClient,
		workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_WORD2PDF.String(),
		worker.Options{
			MaxConcurrentActivityExecutionSize: 1,
		},
	)

	ctrl.pdRemoverWorker = worker.New(
		ctrl.workerClient,
		workflows_pb.WorkflowsQueue_WORKFLOWS_QUEUE_PD_REMOVER.String(),
		worker.Options{
			MaxConcurrentActivityExecutionSize: 1,
		},
	)

	ctrl.defaultWorker.RegisterWorkflowWithOptions(ctrl.WorkflowProcessResumeFile, workflow.RegisterOptions{
		Name: workflows_pb.Workflow_WORKFLOW_PROCESS_RESUME_FILE.String(),
	})

	ctrl.defaultWorker.RegisterActivityWithOptions(ctrl.activityBackendGrpcCall.PatchCandidateResume, activity.RegisterOptions{
		Name: ActivityPatchCandidateResume,
	})

	ctrl.ocrWorker.RegisterActivityWithOptions(ctrl.activityReadTextFromPdf.ReadTextFromPDF, activity.RegisterOptions{
		Name: ActivityReadTextFromPDF,
	})

	ctrl.storageWorker.RegisterActivityWithOptions(ctrl.activityStorage.GetFile, activity.RegisterOptions{
		Name: ActivityGetFile,
	})

	ctrl.word2pdfWorker.RegisterActivityWithOptions(ctrl.activityConvertWordToPDF.ConvertWordToPDF, activity.RegisterOptions{
		Name: ActivityConvertWordToPDF,
	})

	ctrl.pdRemoverWorker.RegisterActivityWithOptions(ctrl.activityPDRemover.AnonymizeText, activity.RegisterOptions{
		Name: ActivityAnonymizeText,
	})
}

func (ctrl *Controller) StartWorkers() {
	if ctrl.defaultWorker != nil {
		ctrl.defaultWorker.Start()
		ctrl.ocrWorker.Start()
		ctrl.storageWorker.Start()
		ctrl.word2pdfWorker.Start()
		ctrl.pdRemoverWorker.Start()
	}
}

func (ctrl *Controller) StopWorkers() {
	if ctrl.defaultWorker != nil {
		ctrl.defaultWorker.Stop()
		ctrl.ocrWorker.Stop()
		ctrl.storageWorker.Stop()
		ctrl.word2pdfWorker.Stop()
		ctrl.pdRemoverWorker.Stop()
	}
}
