package native

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pbrpc"
	"google.golang.org/grpc"
)

/**
core/contract/bridge/vm.go中定义的
Native虚拟机的接口的实现
*/

type nativeCreator struct {
	config   *bridge.InstanceCreatorConfig
	listener net.Listener
	pm       *processManager
}

// newNativeCreator 发现Native虚拟机，由XBridge统一管理，initVM时运行
func newNativeCreator(cfg *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	creator := &nativeCreator{
		config: cfg,
	}
	err := os.MkdirAll(cfg.Basedir, 0755)
	if err != nil {
		return nil, err
	}

	listenAddr, err := creator.startRpcServer(cfg.SyscallService)
	if err != nil {
		return nil, err
	}

	pm, err := newProcessManager(cfg.VMConfig.(*contract.NativeConfig), cfg.Basedir, listenAddr)
	if err != nil {
		return nil, err
	}
	creator.pm = pm

	return creator, nil
}

// startRpcServer 将SyscallService注册成为一个Grpc服务
func (n *nativeCreator) startRpcServer(service *bridge.SyscallService) (string, error) {
	// 1.监听端口号
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	n.listener = listener
	// 2.实例化给RPC实例
	rpcServer := grpc.NewServer()
	// 3.注册服务
	pbrpc.RegisterSyscallServer(rpcServer, service)

	port := listener.Addr().(*net.TCPAddr).Port
	// 4.启动rpc服务
	go rpcServer.Serve(listener)
	chainAddr := chainAddrHost
	if n.config.VMConfig.(*contract.NativeConfig).Docker.Enable {
		chainAddr = chainAddrDocker
	}
	addr := fmt.Sprintf("tcp://%s:%d", chainAddr, port)

	return addr, nil
}

// CreateInstance 创建虚拟机实例
func (n *nativeCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	process, err := n.pm.GetProcess(ctx.ContractName, cp)
	if err != nil {
		return nil, err
	}
	return newNativeVmInstance(ctx, process), nil
}

// RemoveCache 清除虚拟机缓存
func (n *nativeCreator) RemoveCache(name string) {

}

type nativeVmInstance struct {
	ctx     *bridge.Context
	process *contractProcess
}

func newNativeVmInstance(ctx *bridge.Context, process *contractProcess) *nativeVmInstance {
	return &nativeVmInstance{
		ctx:     ctx,
		process: process,
	}
}

// Exec 执行合约调用
func (i *nativeVmInstance) Exec() error {
	request := &pb.NativeCallRequest{
		Ctxid: i.ctx.ID,
	}
	_, err := i.process.RpcClient().Call(context.TODO(), request)
	return err
}

func (i *nativeVmInstance) ResourceUsed() contract.Limits {
	return contract.Limits{
		XFee: 1,
	}
}

func (i *nativeVmInstance) Release() {

}

func (i *nativeVmInstance) Abort(msg string) {
}

// init 注册Native虚拟机
func init() {
	bridge.Register(bridge.TypeNative, "native", newNativeCreator)
}
