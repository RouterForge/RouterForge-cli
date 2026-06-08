package engine

import "fmt"

type ErrorCode int

const (
	ErrUnknown ErrorCode = iota
	ErrNotFound
	ErrInvalidInput
	ErrPermissionDenied
	ErrTimeout
	ErrAPIError
	ErrNetworkError
	ErrAgentError
	ErrPhaseTransition
	ErrTaskFailed
	ErrStorageError
)

func (e ErrorCode) String() string {
	switch e {
	case ErrNotFound:
		return "not_found"
	case ErrInvalidInput:
		return "invalid_input"
	case ErrPermissionDenied:
		return "permission_denied"
	case ErrTimeout:
		return "timeout"
	case ErrAPIError:
		return "api_error"
	case ErrNetworkError:
		return "network_error"
	case ErrAgentError:
		return "agent_error"
	case ErrPhaseTransition:
		return "phase_transition"
	case ErrTaskFailed:
		return "task_failed"
	case ErrStorageError:
		return "storage_error"
	default:
		return "unknown"
	}
}

type RouterForgeError struct {
	Code    ErrorCode
	Message string
	AgentID string
	TaskID  string
	Phase   string
	Cause   error
}

func (e *RouterForgeError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e *RouterForgeError) Unwrap() error {
	return e.Cause
}

func NewError(code ErrorCode, msg string) *RouterForgeError {
	return &RouterForgeError{Code: code, Message: msg}
}

func WrapError(code ErrorCode, msg string, cause error) *RouterForgeError {
	return &RouterForgeError{Code: code, Message: msg, Cause: cause}
}

func AgentError(agentID, taskID, msg string, cause error) *RouterForgeError {
	return &RouterForgeError{
		Code:    ErrAgentError,
		Message: msg,
		AgentID: agentID,
		TaskID:  taskID,
		Cause:   cause,
	}
}

func PhaseError(phase string, msg string, cause error) *RouterForgeError {
	return &RouterForgeError{
		Code:    ErrPhaseTransition,
		Message: msg,
		Phase:   phase,
		Cause:   cause,
	}
}
