package bridge

/**
XBridge抽象出来的一个通用的合约虚拟机接口
*/
import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
)

// InstanceCreatorConfig 虚拟机配置信息
type InstanceCreatorConfig struct {
	Basedir        string
	SyscallService *SyscallService
	// VMConfig is the config of vm driver
	VMConfig VMConfig
}

type VMConfig interface {
	DriverName() string // 驱动名称，可以default(kernel)、xvm、native、evm
	IsEnable() bool     // 是否开启
}

// NewInstanceCreatorFunc 根据虚拟机配置信息返回虚拟机
type NewInstanceCreatorFunc func(config *InstanceCreatorConfig) (InstanceCreator, error)

// ContractCodeProvider 主要提供合约代码与合约描述
type ContractCodeProvider interface {
	// GetContractCodeDesc 获取合约描述
	GetContractCodeDesc(name string) (*protos.WasmCodeDesc, error)
	// GetContractCode 获取合约代码
	GetContractCode(name string) ([]byte, error)
	// GetContractAbi 获取合约ABI，针对EVM合约
	GetContractAbi(name string) ([]byte, error)
	// GetContractCodeFromCache 从缓存中获取合约代码
	GetContractCodeFromCache(name string) ([]byte, error)
	// GetContractAbiFromCache 从缓存中获取合约ABI，针对EVM合约
	GetContractAbiFromCache(name string) ([]byte, error)
}

// InstanceCreator XuperChain对合约虚拟机的约束
// 目前共有四种类型的虚拟机实现：
// KernelInstance 用于Kernel合约的执行
// XVMInstance 用于WASM合约的执行
// NativeInstance 用于Native原生合约的执行
// EVMInstance 用于EVM合约的执行
type InstanceCreator interface {
	// CreateInstance 创建一个虚拟机实例，每次调用合约时创建一个实例
	CreateInstance(ctx *Context, cp ContractCodeProvider) (Instance, error)
	// RemoveCache 清除有关缓存，释放资源，只有XVM虚拟机需要释放缓存
	RemoveCache(name string)
}

// Instance 虚拟机实例的接口约束
type Instance interface {
	Exec() error                   // 执行合约调用
	ResourceUsed() contract.Limits // 获取本次合约调用的资源消耗
	Release()                      // 合约执行完毕，释放有关资源
	Abort(msg string)              // 合约执行异常，中止执行
}
