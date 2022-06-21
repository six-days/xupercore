package native

import (
	"time"
)

/**
启动与停止合约进程的接口约束
*/
const (
	pingTimeoutSecond = 2
)

// Process is the container of running contract
type Process interface {
	// Start 启动Native code进程
	Start() error

	// Stop 停止进程，如果在超时时间内进程没有退出则强制杀死进程
	Stop(timeout time.Duration) error
}
