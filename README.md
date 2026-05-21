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
    f := factory.NewBehaviorTreeFactory()

    f.RegisterSimpleAction("Hello", func(core.TreeNode) core.NodeStatus {
        fmt.Println("Hello, BT!")
        return core.SUCCESS
    }, core.PortsList{})

    tree, _ := f.CreateTreeFromText(`
        <root BTCPP_format="4">
            <BehaviorTree ID="Main">
                <Hello/>
            </BehaviorTree>
        </root>`, nil)

    tree.TickOnce()
    // Output: Hello, BT!
}
```

## 安装

```bash
go get github.com/actfuns/behaviortree
```

## 完整使用示例

### 自定义动作节点

```go
f.RegisterSimpleAction("Say", func(n core.TreeNode) core.NodeStatus {
    // 从黑板读取端口值
    var msg string
    if err := n.GetInput("message", &msg); err != nil {
        return core.FAILURE
    }
    fmt.Println(msg)

    // 写回黑板
    n.SetOutput("output", "done")
    return core.SUCCESS
}, core.PortsList{
    "message": core.NewPortInfo(core.INPUT),
    "output":  core.NewPortInfo(core.OUTPUT),
})
```

```xml
<Say message="{my_text}" output="{result}"/>
```

### 带 RUNNING 的节点（异步动作）

```go
type CountNode struct {
    core.StatefulActionNode
    counter int
}

func (n *CountNode) OnStart() core.NodeStatus {
    n.counter = 5
    return core.RUNNING
}

func (n *CountNode) OnRunning() core.NodeStatus {
    n.counter--
    fmt.Printf("count: %d\n", n.counter)
    if n.counter <= 0 {
        return core.SUCCESS
    }
    return core.RUNNING
}
```

### 完整 XML 树

```xml
<root BTCPP_format="4">
    <BehaviorTree ID="Main">
        <Sequence>
            <!-- 黑板操作 -->
            <Script code="msg := 'hello'"/>
            <Script code="count := 0"/>

            <!-- 条件判断 -->
            <Inverter>
                <Script code="count >= 5"/>
            </Inverter>

            <!-- 循环直到条件满足 -->
            <Repeat num_cycles="3">
                <SetBlackboard output_key="count" value="count + 1"/>
            </Repeat>

            <!-- 超时保护 -->
            <Timeout msec="500">
                <Sequence>
                    <Sleep msec="1000"/>
                    <Say message="done"/>
                </Sequence>
            </Timeout>

            <!-- 延迟执行 -->
            <Delay delay_msec="200">
                <Say message="{msg}"/>
            </Delay>

            <!-- 子树复用 -->
            <SubTree ID="Sub" input="{count}"/>
        </Sequence>
    </BehaviorTree>
</root>
```

### 主循环驱动

```go
// 嵌入自己的循环（推荐）
func gameLoop(tree *core.Tree) {
    for {
        status := tree.TickOnce()
        // 不阻塞，微秒级返回

        if status == core.SUCCESS {
            break
        }

        // 处理自己的逻辑...
        time.Sleep(16 * time.Millisecond)
    }
}
```

### 编程方式构建树

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
tree.Initialize()
tree.TickOnce()
```

## 架构

### 包一览

| 包 | 职责 |
|---|---|
| [core](core/) | 节点接口、状态机、黑板、端口/类型系统、脚本环境、TimerQueue |
| [factory](factory/) | 工厂实现，注册所有内建节点，管理与生命周期 |
| [control](control/) | 序列、回退、并行、反应式序列、IfThenElse、WhileDoElse、Switch、TryCatch |
| [decorator](decorator/) | 重试、重复、超时、延迟、逆变器、ForceSuccess/Failure、KeepRunning、SubTree |
| [action](action/) | 内建动作：Script、Sleep、SetBlackboard、AlwaysSuccess/Failure |
| [script](script/) | 嵌入式脚本引擎：赋值、算术、比较、三元运算 |
| [xml](xml/) | BTCPP_format="4" XML 解析与序列化 |

### 节点状态机

```
IDLE ── tick ──► RUNNING ── tick ──► SUCCESS / FAILURE
  ▲                    │
  └──── reset ◄────────┘
```

| 状态 | 含义 |
|---|---|
| `IDLE` | 初始状态，尚未执行 |
| `RUNNING` | 执行中，需要持续 tick |
| `SUCCESS` | 执行成功 |
| `FAILURE` | 执行失败 |

## 驱动方式

### Tick 模式

| 函数 | 行为 | 适用场景 |
|---|---|---|
| `TickExactlyOnce()` | 精确 tick 一次 | 嵌入外部循环，完全自己控制节奏 |
| `TickOnce()` | tick 一次 + 处理残留唤醒信号 | 嵌入外部循环，需要及时响应 timer |
| `TickWhileRunning(sleepTime)` | 阻塞循环 tick，直到树完成 | 独立后台 goroutine |

### 嵌入主循环（推荐）

```go
for {
    status := tree.TickOnce()
    // 不阻塞，微秒级返回

    if status.IsCompleted() {
        break
    }

    // 处理输入、渲染等
    time.Sleep(16 * time.Millisecond)
}
```

`TickOnce` vs `TickExactlyOnce`：

```
帧1: TickOnce          → ExecuteTick → RUNNING → WaitFor(0) 无 signal → 返回
帧2: timer 触发！signal → ───┐
帧2: TickOnce          → ExecuteTick → RUNNING
                           → WaitFor(0) 发现 signal
                           → 立即重新 ExecuteTick → SUCCESS ← 同一帧
```

`TickOnce` 多一层内层 spin loop，将残缺响应从两帧缩短到一帧。`TickExactlyOnce` 不消费 signal，适合完全自己控制 tick 节奏的场景。

### 后台独立运行

用于树与主线程不共享数据的场景：

```go
go tree.TickWhileRunning(100 * time.Millisecond)
```

内部自带循环，阻塞等待 timer 或超时，由 `WakeUpSignal` 精确唤醒。

### 注意事项

`TickOnce` / `TickExactlyOnce` 与 `TickWhileRunning` **互斥**，同一棵树不能同时在两个 goroutine 中 tick。

## TimerQueue 设计

内建节点（Sleep、Delay、Timeout）使用 TimerQueue 实现延时和超时，后台 timer 精确触发，不依赖 tick 间隔：

```go
// timer callback 只设标志 + 发唤醒，不直接操作节点状态
n.timerID = n.TimerQueue().Add(
    core.DurationFromMS(n.msec),
    func(aborted bool) {
        n.mu.Lock()
        n.childHalted = true
        n.mu.Unlock()
        n.EmitWakeUpSignal()
    },
)
```

- **TimerQueue**：后台 goroutine + 优先堆，定时触发回调
- **WakeUpSignal**：buffered channel（1），发送和轮询均非阻塞
- **全局默认队列**：脱离 Tree 测试时自动使用

## 端口系统

节点通过端口声明输入/输出，XML 中通过 `{}` 语法映射到黑板键。

```go
f.RegisterNodeType("SetMessage", core.PortsList{
    "message": core.NewPortInfo(core.INPUT),
}, func(name string, config core.NodeConfig) core.TreeNode {
    return NewSetMessage(name, config)
}, core.Action)
```

```xml
<SetMessage message="{my_text}"/>
```

端口方向：`INPUT` / `OUTPUT` / `INOUT`。

## 黑板（Blackboard）

```xml
<Sequence>
    <Script code="counter := 0"/>
    <SetBlackboard output_key="counter" value="counter + 1"/>
    <Say message="{counter}"/>
</Sequence>
```

## 脚本表达式

用在 `<Script>`、`_skipIf`、`_successIf`、`_failureIf` 等属性中。

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
| `_skipIf="expr"` | 表达式为真时节点跳过 |
| `_while="expr"` | 表达式为真才允许节点运行 |
| `_onSuccess="script"` | 节点成功时执行脚本 |
| `_onFailure="script"` | 节点失败时执行脚本 |
| `_onHalted="script"` | 节点被中断时执行脚本 |

## 子树与端口重映射

```xml
<BehaviorTree ID="Sub">
    <Script code="msg := '{input}'"/>
</BehaviorTree>

<BehaviorTree ID="Main">
    <SubTree ID="Sub" input="{my_variable}"/>
</BehaviorTree>
```

## 测试

```bash
go test github.com/actfuns/behaviortree/...
go test -race github.com/actfuns/behaviortree/...
```

## 参考

基于 [BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) **v4.9.0**（commit `e6754eeb`）。
