// Package fxboot contains fx bootstrapping
package fxboot

import (
	"go.uber.org/fx"
)

// ProvidingID - type for providing id
type ProvidingID int

const (
	// ProvidingAppID - app id
	ProvidingAppID ProvidingID = iota

	// ProvidingIDFXTimeouts - fx timeouts
	ProvidingIDFXTimeouts

	// ProvidingIDConfig - app config
	ProvidingIDConfig

	// ProvidingIDLogger - logger
	ProvidingIDLogger

	// ProvidingIDFXLogger - fx logger
	ProvidingIDFXLogger

	// ProvidingIDTemporalWorker - temporal worker
	ProvidingIDTemporalWorker

	// ProvidingIDWorkflowsController - workflows controller
	ProvidingIDWorkflowsController

	// ProvidingIDWorkflowsActivies - workflows activities
	ProvidingIDWorkflowsActivies

	// ProvidingIDStorageClient - storage client
	ProvidingIDStorageClient

	// ProvidingIDGRPCBackendPrivateConnection - grpc backend connection
	ProvidingIDGRPCBackendPrivateConnection

	// ProvidingIDGRPCBackendClient - grpc backend client
	ProvidingIDGRPCBackendClient
)

// OptionsMap - options map item with providing and invokes elements
type OptionsMap struct {
	Providing map[ProvidingID]fx.Option
	Invokes   []fx.Option
}

// OptionsMapToSlice - convert options map to slice for fx bootstrapping
func OptionsMapToSlice(optionsMap OptionsMap) []fx.Option {
	result := make([]fx.Option, 0)

	for _, option := range optionsMap.Providing {
		result = append(result, option)
	}

	result = append(result, optionsMap.Invokes...)

	return result
}
