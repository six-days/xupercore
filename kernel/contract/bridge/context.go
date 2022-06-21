package bridge

import (
	"sync"

	"github.com/xuperchain/xupercore/lib/logs"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/protos"
)

// Context 保存了合约的运行参数、执行沙盒、输出结果、事件、日志等
// 所有的系统调用产生的状态保存在这里
// 用于隔离多个合约的执行，也便于合约的并发执行。
// 所有的Context由XBridge管理，虚拟机只需要关注无状态的合约执行
type Context struct {
	ID     int64
	Module string
	// 合约名字
	ContractName string

	ResourceLimits contract.Limits

	State contract.StateSandbox

	Args map[string][]byte

	Method string

	Initiator string

	Caller string

	AuthRequire []string

	CanInitialize bool

	Core contract.ChainCore

	TransferAmount string

	Instance Instance

	Logger logs.Logger

	// 二级合约调用资源限制，合约调用合约
	SubResourceUsed contract.Limits

	// Contract being called
	// set by bridge to check recursive contract call
	ContractSet map[string]bool

	// 合约事件
	Events []*protos.ContractEvent

	// 输出结果
	Output *pb.Response

	// Read from cache
	ReadFromCache bool
}

// DiskUsed 合约磁盘占用
func (c *Context) DiskUsed() int64 {
	size := int64(0)
	wset := c.State.RWSet().WSet
	for _, w := range wset {
		size += int64(len(w.GetKey()))
		size += int64(len(w.GetValue()))
	}
	return size
}

// ExceedDiskLimit 是否超过磁盘的最大资源限制
func (c *Context) ExceedDiskLimit() bool {
	size := c.DiskUsed()
	return size > c.ResourceLimits.Disk
}

// ResourceUsed returns the resource used by context
func (c *Context) ResourceUsed() contract.Limits {
	// 历史原因kernel合约只计算虚拟机的资源消耗
	if c.Module == string(TypeKernel) {
		return c.Instance.ResourceUsed()
	}
	var total contract.Limits
	total.Add(c.Instance.ResourceUsed()).Add(c.SubResourceUsed)
	// 事件资源占用
	total.Add(eventsResourceUsed(c.Events))
	total.Disk += c.DiskUsed()
	return total
}

// GetInitiator 返回合约发起人
func (c *Context) GetInitiator() string {
	return c.Initiator
}

// GetAuthRequire 返回签名
func (c *Context) GetAuthRequire() []string {
	return c.AuthRequire
}

// ContextManager 用于管理产生和销毁Context，其作用包括：
// 维护全局递增的ContextId
// 按需进行Context的创建和销毁
// 保存所有合约调用的状态
// 根据ContextId，返回上下文信息
type ContextManager struct {
	// 保护如下两个变量
	// 合约进行系统调用以及合约执行会并发访问ctxs
	ctxlock sync.Mutex
	ctxid   int64
	ctxs    map[int64]*Context
}

// NewContextManager 系统初始化时构造唯一一个ContextManager对象
func NewContextManager() *ContextManager {
	return &ContextManager{
		ctxs: make(map[int64]*Context),
	}
}

// Context 根据context的id返回当前运行当前合约的上下文
func (n *ContextManager) Context(id int64) (*Context, bool) {
	n.ctxlock.Lock()
	defer n.ctxlock.Unlock()
	ctx, ok := n.ctxs[id]
	return ctx, ok
}

// MakeContext 递增一个ContextId并分配一个Context
func (n *ContextManager) MakeContext() *Context {
	n.ctxlock.Lock()
	defer n.ctxlock.Unlock()
	n.ctxid++
	ctx := new(Context)
	ctx.ID = n.ctxid
	n.ctxs[ctx.ID] = ctx
	return ctx
}

// DestroyContext 一定要在合约执行完毕（成功或失败）进行销毁
func (n *ContextManager) DestroyContext(ctx *Context) {
	n.ctxlock.Lock()
	defer n.ctxlock.Unlock()
	delete(n.ctxs, ctx.ID)
}
