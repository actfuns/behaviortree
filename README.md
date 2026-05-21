# BehaviorTree — Go 行为树引擎

[BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) v4.x 的 Go 移植。405+ 测试用例覆盖全部核心功能。

## 快速开始

```go
package main

import (
    "fmt"
    "time"

    "github.com/actfuns/behaviortree/core"
    "github.com/actfuns/behaviortree/factory"
)

func main() {
    f := factory.NewBehaviorTreeFactory()

    // 注册动作节点
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

## 架构

### 包一览

| 包 | 职责 |
|---|---|
| [core](core/) | 节点接口、状态机、黑板、端口/类型系统、脚本环境、TimerQueue、WakeUpSignal |
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
| `RUNNING` | 执行中，需要继续 tick |
| `SUCCESS` | 执行成功 |
| `FAILURE` | 执行失败 |

## 驱动行为树

### Tick 模式

库提供三种驱动树的方式，适用于不同的使用场景：

| 函数 | 行为 | 适用场景 |
|---|---|---|
| `TickExactlyOnce()` | 精确 tick 一次，不处理唤醒信号 | 嵌入外部循环，自己控制节奏 |
| `TickOnce()` | tick 一次 + 处理残留唤醒信号 | 嵌入外部循环，需要及时响应 timer |
| `TickWhileRunning()` | 阻塞循环 tick，直到树完成或停止 | 独立后台 goroutine |

### 嵌入主循环（推荐）

用于游戏循环、机器人控制框架等已有主循环的场景：

```go
// 你自己的游戏/控制循环
for {
    status := tree.TickOnce()
    // TikOnce 不阻塞，微秒级返回
    // 如果有 timer 在上一帧触发，会在同一帧内立即响应

    if status == core.SUCCESS {
        fmt.Println("树执行完成")
    }

    time.Sleep(16 * time.Millisecond) // ~60 FPS
}
```

`TickOnce` 与 `TickExactlyOnce` 的区别：

```
帧1: tree.TickExactlyOnce → ExecuteTick → RUNNING
                             timer 在这帧触发
帧2: tree.TickExactlyOnce → ExecuteTick → RUNNING
                             忘了检查 signal
帧3: tree.TickExactlyOnce → ExecuteTick → SUCCESS

帧1: tree.TickOnce         → ExecuteTick → RUNNING
                             没有 signal，返回
帧2: tree.TickOnce         → ExecuteTick → RUNNING
                             → WaitFor(0) 发现 signal
                             → 立即重新 ExecuteTick → SUCCESS
                             同一帧内返回
```

`TickOnce` 多了一个内层 spin loop，用于 drain 上一帧到这一帧之间积累的 timer signal，将响应速度从两帧缩短到一帧。

### 后台独立运行

用于树不与主线程共享数据的场景：

```go
// 后台 goroutine，每 100ms tick 一次
go tree.TickWhileRunning(100 * time.Millisecond)
```

`TickWhileRunning` 内部自带循环和阻塞等待：

```go
for status == IDLE || status == RUNNING {
    status = root.ExecuteTick()
    // 内层 spin loop：drain 残留 signal
    for status == RUNNING && wakeUp.WaitFor(0) {
        status = root.ExecuteTick()
    }
    if status.IsCompleted() {
        root.ResetStatus()
    }
    if status == RUNNING {
        wakeUp.WaitFor(sleepTime)  // 阻塞等待或超时
    }
}
```

`WaitFor(sleepTime)` 是阻塞的，但由 timer callback 精确唤醒，不浪费 CPU。

### 注意事项

`TickOnce` 和 `TickWhileRunning` **互斥**，同一棵树不能同时被两个 goroutine 调用。前者为主线程单帧调用设计，后者为后台独立 goroutine 设计。

## 实际场景

### 延时执行（DelayNode）

```xml
<BehaviorTree ID="Main">
    <Delay delay_msec="500">
        <AlwaysSuccess/>
    </Delay>
</BehaviorTree>
```

首次 tick 启动 500ms 后台 timer，期间返回 RUNNING。timer 触发时通过 `EmitWakeUpSignal` 唤醒树：

```
tick 1: Delay 启动 500ms timer → RUNNING
tick 2: timer 未到期 → RUNNING
        ...
500ms 后 timer 触发 → 发 signal → 树被唤醒
tick N: Delay 发现 timer 到期 → 执行子节点 → SUCCESS
```

无需 `TickWhileRunning` 也可工作（signal 通过 `TickOnce` 的 spin loop 消费）。

### 超时保护（TimeoutNode）

```xml
<BehaviorTree ID="Main">
    <Timeout msec="200">
        <Sleep msec="5000"/>  <!-- 5秒任务 -->
    </Timeout>
</BehaviorTree>
```

200ms 后 timer 触发，设置 childHalted 标志，发送 wake-up。下一次 tick 时 TimeoutNode 检测到超时，halt 子节点，返回 FAILURE。

### 定时任务（SleepNode）

```go
f.RegisterSimpleAction("Work", func(n core.TreeNode) core.NodeStatus {
    // 执行工作任务
    time.Sleep(20 * time.Millisecond) // 模拟耗时
    return core.SUCCESS
}, core.PortsList{})
```

```xml
<BehaviorTree ID="Main">
    <Sequence>
        <Sleep msec="1000"/>
        <Work/>
    </Sequence>
</BehaviorTree>
```

Sleep 精确等待 1 秒，tick 期间返回 RUNNING。timer 触发后通过 wake-up 信号立即触发后续节点，无需等待下一轮 tick 间隔。

## TimerQueue 设计

内建节点（Sleep、Delay、Timeout）使用 TimerQueue 实现延时和超时：

- **TimerQueue**：后台 goroutine + 优先堆，定时触发回调
- **WakeUpSignal**：buffered channel（1），`Emit()` 非阻塞发送，`WaitFor(0)` 非阻塞轮询
- **全局默认队列**：脱离 Tree 单独测试时自动使用默认队列

后台 timer 回调只设置标志位 + 发 wake-up，不直接操作节点状态，避免竞态：

```go
n.timerID = n.TimerQueue().Add(
    core.DurationFromMS(n.msec),
    func(aborted bool) {
        n.mu.Lock()
        n.childHalted = true        // 只设标志
        n.mu.Unlock()
        n.EmitWakeUpSignal()        // 发唤醒信号
    },
)
```

## 配置行为树

### XML 方式（推荐）

```go
factory.RegisterBehaviorTreeFromText(xmlText)
tree, _ := factory.CreateTree("MainTree", blackboard)
```

### 编程方式

```go
package main

import (
    "github.com/actfuns/behaviortree/action"
    "github.com/actfuns/behaviortree/control"
    "github.com/actfuns/behaviortree/core"
)

func main() {
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
}
```

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

端口方向：`INPUT` / `OUTPUT` / `INOUT`。字符串作为通用供体可自动转换为数值类型。

## 黑板（Blackboard）

黑板是节点间共享数据的机制：

```xml
<root BTCPP_format="4">
    <BehaviorTree ID="Main">
        <Sequence>
            <Script code="counter := 0"/>
            <SetBlackboard output_key="counter" value="counter + 1"/>
            <Say message="{counter}"/>
        </Script>
    </Sequence>
</BehaviorTree>
```

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
go test -race github.com/actfuns/behaviortree/...
```

407 测试覆盖：

| 包 | 测试文件 | 测试函数 |
|---|---|---|
| core | 14 | 185 |
| control | 3 | 97 |
| decorator | 3 | 37 |
| factory | 2 | 29 |
| xml | 2 | 29 |
| script | 2 | 22 |
| action | 3 | 8 |

## 参考

基于 [BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) **v4.9.0**（commit `e6754eeb`），分支 `BehaviorTree.CPP`。后续新增功能时，请对齐该版本的 API 和行为。
