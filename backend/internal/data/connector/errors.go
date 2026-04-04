package connector

import (
	"errors"
	"fmt"
)

const (
	ErrorCodeConnectionFailed      = "CONNECTION_FAILED"
	ErrorCodeConnectionTimeout     = "CONNECTION_TIMEOUT"
	ErrorCodeAuthenticationFailed  = "AUTHENTICATION_FAILED"
	ErrorCodeSchemaDiscoveryFailed = "SCHEMA_DISCOVERY_FAILED"
	ErrorCodeQueryFailed           = "QUERY_FAILED"
	ErrorCodeQueryTimeout          = "QUERY_TIMEOUT"
	ErrorCodeUnsupportedOperation  = "UNSUPPORTED_OPERATION"
	ErrorCodeConfigurationError    = "CONFIGURATION_ERROR"
	ErrorCodeDriverError           = "DRIVER_ERROR"
	ErrorCodePermissionDenied      = "PERMISSION_DENIED"
)

type ConnectorError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Type      string `json:"connector_type"`
	Operation string `json:"operation"`
	Cause     error  `json:"-"`
}

func (e *ConnectorError) Error() string {
	return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Operation, e.Code, e.Message)
}

func (e *ConnectorError) Unwrap() error {
	return e.Cause
}

func newConnectorError(connectorType, operation, code, message string, cause error) error {
	return &ConnectorError{
		Code:      code,
		Message:   message,
		Type:      connectorType,
		Operation: operation,
		Cause:     cause,
	}
}

func AsConnectorError(err error, target **ConnectorError) bool {
	if target == nil {
		return false
	}
	return errors.As(err, target)
}
