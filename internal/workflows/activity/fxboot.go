package activity

import (
	"github.com/neurochar/workflows/internal/workflows/activity/backend_grpc_call"
	"github.com/neurochar/workflows/internal/workflows/activity/personal_data_remover"
	"github.com/neurochar/workflows/internal/workflows/activity/read_text_from_pdf"
	"github.com/neurochar/workflows/internal/workflows/activity/storage"
	"github.com/neurochar/workflows/internal/workflows/activity/word2pdf"
	"go.uber.org/fx"
)

// FxModule - fx module
var FxModule = fx.Options(
	backend_grpc_call.FxModule,
	read_text_from_pdf.FxModule,
	storage.FxModule,
	word2pdf.FxModule,
	personal_data_remover.FxModule,
)
