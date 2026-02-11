package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNil           = errors.New("redisx: nil value")
	ErrKeyNotFound   = errors.New("redisx: key not found")
	ErrLockNotHeld   = errors.New("redisx: lock not held")
	ErrLockExpired   = errors.New("redisx: lock expired")
	ErrLockConflict  = errors.New("redisx: lock conflict")
	ErrConfigNil     = errors.New("redisx: config is nil")
	ErrConfigMode    = errors.New("redisx: invalid config mode")
	ErrConfigSingleAddr    = errors.New("redisx: single config addr is required")
	ErrConfigSentinelNil   = errors.New("redisx: sentinel config is required")
	ErrConfigSentinelMasterName = errors.New("redisx: sentinel master name is required")
	ErrConfigSentinelAddrs = errors.New("redisx: sentinel addrs are required")
	ErrConfigClusterAddrs  = errors.New("redisx: cluster addrs are required")
	ErrConfigMultiMasterMasters = errors.New("redisx: multi-master masters are required")
	ErrClientClosed   = errors.New("redisx: client is closed")
	ErrClientNotReady = errors.New("redisx: client is not ready")
	ErrNoAvailableNode = errors.New("redisx: no available redis node")
	ErrRetryExhausted = errors.New("redisx: retry exhausted")
)

type ConfigError struct {
	Field   string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("redisx: config error - field '%s': %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: config error - field '%s': %s", e.Field, e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

func NewConfigError(field, message string, err error) *ConfigError {
	return &ConfigError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

type OperationError struct {
	Operation string
	Key       string
	Message   string
	Err       error
}

func (e *OperationError) Error() string {
	if e.Key != "" {
		if e.Err != nil {
			return fmt.Sprintf("redisx: operation '%s' on key '%s': %s: %v", e.Operation, e.Key, e.Message, e.Err)
		}
		return fmt.Sprintf("redisx: operation '%s' on key '%s': %s", e.Operation, e.Key, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("redisx: operation '%s': %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: operation '%s': %s", e.Operation, e.Message)
}

func (e *OperationError) Unwrap() error {
	return e.Err
}

func NewOperationError(operation, key, message string, err error) *OperationError {
	return &OperationError{
		Operation: operation,
		Key:       key,
		Message:   message,
		Err:       err,
	}
}

type LockError struct {
	LockKey  string
	Message  string
	Err      error
}

func (e *LockError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("redisx: lock error on key '%s': %s: %v", e.LockKey, e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: lock error on key '%s': %s", e.LockKey, e.Message)
}

func (e *LockError) Unwrap() error {
	return e.Err
}

func NewLockError(lockKey, message string, err error) *LockError {
	return &LockError{
		LockKey: lockKey,
		Message: message,
		Err:     err,
	}
}

type CacheError struct {
	CacheType string
	Key       string
	Message   string
	Err       error
}

func (e *CacheError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("redisx: cache error - type '%s', key '%s': %s: %v", e.CacheType, e.Key, e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: cache error - type '%s', key '%s': %s", e.CacheType, e.Key, e.Message)
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

func NewCacheError(cacheType, key, message string, err error) *CacheError {
	return &CacheError{
		CacheType: cacheType,
		Key:       key,
		Message:   message,
		Err:       err,
	}
}

type ConnectionError struct {
	Node    string
	Message string
	Err     error
}

func (e *ConnectionError) Error() string {
	if e.Node != "" {
		if e.Err != nil {
			return fmt.Sprintf("redisx: connection error - node '%s': %s: %v", e.Node, e.Message, e.Err)
		}
		return fmt.Sprintf("redisx: connection error - node '%s': %s", e.Node, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("redisx: connection error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: connection error: %s", e.Message)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

func NewConnectionError(node, message string, err error) *ConnectionError {
	return &ConnectionError{
		Node:    node,
		Message: message,
		Err:     err,
	}
}

type RetryError struct {
	Attempts int
	Message  string
	Err      error
}

func (e *RetryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("redisx: retry error after %d attempts: %s: %v", e.Attempts, e.Message, e.Err)
	}
	return fmt.Sprintf("redisx: retry error after %d attempts: %s", e.Attempts, e.Message)
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

func NewRetryError(attempts int, message string, err error) *RetryError {
	return &RetryError{
		Attempts: attempts,
		Message:  message,
		Err:      err,
	}
}

func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}

func IsOperationError(err error) bool {
	_, ok := err.(*OperationError)
	return ok
}

func IsLockError(err error) bool {
	_, ok := err.(*LockError)
	return ok
}

func IsCacheError(err error) bool {
	_, ok := err.(*CacheError)
	return ok
}

func IsConnectionError(err error) bool {
	_, ok := err.(*ConnectionError)
	return ok
}

func IsRetryError(err error) bool {
	_, ok := err.(*RetryError)
	return ok
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	return IsConnectionError(err) || IsRetryError(err) || errors.Is(err, ErrNoAvailableNode)
}
