# BehaviorTree — Go 行为树引擎

[BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) v4.x 的 Go 移植。405+ 测试用例覆盖全部核心功能。

## 快速开始

```go
package main

import (
    "fmt"

    "github.com/actfuns/behaviortree/core"
    "github.com/actfuns/behaviortree/factory"
)

func main() {
    factory := factory.NewBehaviorTreeFactory()
    factory.RegisterSimpleAction("Say", func(core.TreeNode) core.NodeStatus {
        fmt.Println("Hello, BT!")
        return core.SUCCESS
    }, core.PortsList{})

    tree, _ := factory.CreateTreeFromText(`
        <root BTCPP_format="4">
            <BehaviorTree ID="Main">
                <Say/>
            </BehaviorTree>
        </root>`, nil)

    tree.TickWhileRunning(0)
    // Output: Hello, BT!
}
```

## 安装

```bash
go get github.com/actfuns/behaviortree
```

## 架构

### 包一览

| 包 | 职责 |
|---|---|
| [core](core/) | 节点接口、状态机、黑板、端口/类型系统、脚本环境 |
| [factory](factory/) | 工厂实现，注册所有内建节点，管理与生命周期 |
| [control](control/) | 序列、回退、并行、反应式序列、IfThenElse、WhileDoElse、Switch、TryCatch |
| [decorator](decorator/) | 重试、重复、超时、延迟、逆变器、ForceSuccess/Failure、KeepRunning、SubTree、前置条件 |
| [action](action/) | 内建动作：Script、Sleep、SetBlackboard、AlwaysSuccess/Failure、EntryUpdated |
| [script](script/) | 嵌入式脚本引擎：赋值、算术、比较、三元运算 |
| [xml](xml/) | BTCPP_format="4" XML 解析与序列化 |

### 节点类型

| 类型 | 说明 |
|---|---|
| **Action** | 执行一个具体动作，返回 SUCCESS / FAILURE / RUNNING |
| **Condition** | 检查条件，返回 SUCCESS / FAILURE |
| **Control** | 控制子节点的执行顺序和策略 |
| **Decorator** | 包装单个子节点，修改其行为 |
| **SubTree** | 引用另一棵行为树 |

### 节点状态机

```
IDLE ── tick ──► RUNNING ── tick ──► SUCCESS / FAILURE / SKIPPED
  ▲                    │
  └──── reset ◄────────┘
```

| 状态 | 含义 |
|---|---|
| `IDLE` | 初始状态，尚未执行 |
| `RUNNING` | 执行中，需要继续 tick |
| `SUCCESS` | 执行成功 |
| `FAILURE` | 执行失败 |
| `SKIPPED` | 因前置条件跳过 |

### 三种 Tick 模式

- **`TickOnce()`** — 单次 tick，处理唤醒信号，返回顶层状态
- **`TickExactlyOnce()`** — 单次精确 tick（不额外处理唤醒）
- **`TickWhileRunning(timeout)`** — 循环 tick 直到完成或超时

## 配置行为树

### XML 方式（推荐）

```go
factory.RegisterBehaviorTreeFromText(xmlText)
tree, _ := factory.CreateTree("MainTree", blackboard)
```

### 编程方式

```go
bb := core.NewBlackboard(nil)

seq := control.NewSequenceNode("seq", core.NodeConfig{Blackboard: bb})
act := action.NewAlwaysSuccessNode("ok", core.NodeConfig{Blackboard: bb})
seq.AddChild(act)

tree := core.NewTree()
tree.Subtrees = []*core.TreeSubtree{{
    Nodes:      []core.TreeNode{seq},
    Blackboard: bb,
    TreeID:     "Main",
}}
status := tree.TickWhileRunning(0)
```

## 端口系统

节点通过端口声明输入/输出，XML 中通过 `{}` 语法映射到黑板键。

```go
// 声明带端口的节点
_ = factory.RegisterNodeType("SetMessage", core.PortsList{
    "message": core.NewPortInfo(core.INPUT),
}, func(name string, config core.NodeConfig) core.TreeNode {
    return NewSetMessage(name, config)
}, core.Action)
```

```xml
<!-- XML 中绑定黑板变量 -->
<SetMessage message="{my_text}"/>
```

端口方向：`INPUT` / `OUTPUT` / `INOUT`。字符串作为通用供体可自动转换为数值类型。

## 脚本表达式

用在 `<Script>`、`_skipIf`、`_successIf`、`_failureIf`、`_onSuccess` 等属性中。

```
counter := counter + 1
msg == "done" ? "finished" : "working"
x > 0 && y < 10
```

## 前置/后置条件

| 属性 | 触发条件 |
|---|---|
| `_successIf="expr"` | 表达式为真时节点直接返回 SUCCESS |
| `_failureIf="expr"` | 表达式为真时节点直接返回 FAILURE |
| `_skipIf="expr"` | 表达式为真时节点跳过（SKIPPED） |
| `_while="expr"` | 表达式为真才允许节点运行 |
| `_onSuccess="script"` | 节点成功时执行脚本 |
| `_onFailure="script"` | 节点失败时执行脚本 |
| `_onHalted="script"` | 节点被中断时执行脚本 |

## 子树与端口重映射

```xml
<BehaviorTree ID="Sub">
    <Sequence>
        <Script code="msg := '{input}'"/>
    </Sequence>
</BehaviorTree>

<BehaviorTree ID="Main">
    <SubTree ID="Sub" input="{my_variable}"/>
</BehaviorTree>
```

自动重映射：`_autoremap="true"` 自动将同名字段透传到子树。

## 测试

```bash
go test github.com/actfuns/behaviortree/...
```

407 测试覆盖：

| 包 | 测试文件 | 测试函数 |
|---|---|---|
| core | 14 个 | 185 |
| control | 3 个 | 97 |
| decorator | 3 个 | 37 |
| factory | 2 个 | 29 |
| xml | 2 个 | 29 |
| script | 2 个 | 22 |
| action | 3 个 | 8 |

## 参考

基于 [BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) **v4.9.0**（commit `e6754eeb`），分支 `BehaviorTree.CPP`。后续新增功能时，请对齐该版本的 API 和行为。
