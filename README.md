# tke-bridge-agent

## 概览
tke-bridge-agent 会为节点生成 [tke-bridge 配置](./scripts/tke-bridge.conf) ，该配置组合了 [bridge](https://github.com/containernetworking/plugins/tree/master/plugins/main/bridge) 和 [host-local](https://github.com/containernetworking/plugins/tree/master/plugins/ipam/host-local) 插件。
#### 功能：
* 设置节点 `net.bridge.bridge-nf-call-iptables=1`
* 依据节点`.spec.podCIDR`字段生成 tke-bridge [CNI](https://kubernetes.io/docs/concepts/cluster-administration/network-plugins/#cni)配置。
* 在节点`.spec.podCIDR`字段变化时重新生成 tke-bridge [CNI](https://kubernetes.io/docs/concepts/cluster-administration/network-plugins/#cni)配置。

### 部署指引
tke-bridge-agent 通过 daemonset 部署
```$xslt
kubectl create -f https://raw.githubusercontent.com/qyzhaoxun/tke-bridge-agent/master/deploy/v0.0.4/tke-bridge-agent.yaml
```
*注意：Kubelet 网络插件需要设置为 cni (`--network-plugin=cni`,`--cni-config-dir=/etc/cni/net.d` `--cni-bin-dir=/opt/cni/bin`)*


### 开发指引

#### 依赖
- golang 1.10

#### 编译
* `make` 默认执行 `make build` 会构建 Linux 平台二进制文件。
* `make docker-build` 使用 docker 构建 Linux 平台二进制文件。
* `make docker` 会构建 `tke-bridge-agent` 镜像，镜像 tag 取自 `git describe --tags --always --dirty`。
* `make push` 会推送 `tke-bridge-agent` 镜像，镜像 tag 取自 `git describe --tags --always --dirty`。

### 运行参数
`--mtu`
含义：显示指定 MTU 大小。
默认：0，节点已有网卡 MTU 最小值。
变更风险：不会影响已有网卡。
示例：`--mtu=1500`

`--add-rule`
含义：是否添加策略路由 (`from all to <subnet> lookup main pref 1024`)。
默认：添加。
变更风险：***如果节点运行了 tke-route-eni 类型 Pod，可能会导致 tke-route-eni 类型 Pod 和 tke-bridge 类型 Pod 互访失败。***
示例：`--add-rule`

`--cni-conf-dir`
含义：指定生成 tke-bridge.conf 配置路径。
默认：Pod`/host/etc/cni/net.d/multus`路径，对应节点`/etc/cni/net.d/multus`
变更风险：确保能被加载到。
示例：`--cni-conf-dir=/host/etc/cni/net.d/multus`