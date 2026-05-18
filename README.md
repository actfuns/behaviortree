# BT — 行为树库 (Go)

[BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) 的 Go 移植版本，提供构建模块化、响应式 AI 行为树的框架。

## 包结构

| 包 | 说明 |
|---------|-------------|
| [core](core/) | 核心类型：`TreeNode`、`NodeStatus`、`Blackboard`、`BehaviorTreeFactory`、端口系统 |
| [action](action/) | 内置动作节点：`Script`、`Sleep`、`SetBlackboard`、`AlwaysSuccess/Failure` 等 |
| [control](control/) | 控制流节点：`Sequence`、`Fallback`、`Parallel`、`ReactiveSequence`、`IfThenElse`、`WhileDoElse`、`TryCatch`、`Switch` 等 |
| [decorator](decorator/) | 装饰器节点：`Retry`、`Repeat`、`Timeout`、`Delay`、`Inverter`、`ForceSuccess/Failure`、`SubTree`、`Loop` 等 |
| [script](script/) | BT 脚本分词器、解析器和执行器，用于嵌入式条件表达式 |
| [xml](xml/) | 行为树的 XML 序列化和反序���化 |

## 安装

```bash
go get github.com/actfuns/bt
```

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/actfuns/bt/core"
    _ "github.com/actfuns/bt/xml"
)

func main() {
    factory, _ := core.NewBehaviorTreeFactory()

    factory.RegisterSimpleAction("Hello", func(core.TreeNode) core.NodeStatus {
        fmt.Println("Hello, BT!")
        return core.SUCCESS
    }, core.PortsList{})

    tree, _ := factory.CreateTreeFromText(`
        <root BTCPP_format="4">
            <BehaviorTree ID="Main">
                <Hello/>
            </BehaviorTree>
        </root>`, nil)

    tree.TickWhileRunning(0)
}
```

## 主要特性

- **节点类型**: Action（动作）、Condition（条件）、Control（控制）、Decorator（装饰器）、SubTree（子树）
- **端口系统**: 基于 Blackboard 的类型安全数据流，支持 `InputPort`/`OutputPort`/`BidirectionalPort`
- **响应式节点**: `ReactiveSequence`、`ReactiveFallback` — 每次 tick 重新评估条件
- **异步支持**: 节点返回 `RUNNING` 表示需要多次 tick；唤醒信号驱动自动重试
- **XML 加载**: 完整支持 BehaviorTree.CPP `BTCPP_format="4"` XML 解析
- **脚本表达式**: 支持内联脚本编写条件和变量操作
- **三种 tick 模式**: `TickOnce()` / `TickExactlyOnce()` / `TickWhileRunning()`，匹配 C++ API
- **SubTree 支持**: 树组合与端口重映射
- 5 个 C++ 对比测试，验证执行路径一致性

## 节点状态

| 状态 | 含义 |
|--------|---------|
| `IDLE` | 尚未被 tick |
| `RUNNING` | 执行中，需要更多 tick |
| `SUCCESS` | 执行成功 |
| `FAILURE` | 执行失败 |
| `SKIPPED` | 因前置条件被跳过 |

## 运行测试

```bash
go test ./...
```

全部 7 个包测试通过。5 个 [C++ 对比测试](control/cpp_comparison_test.go) 验证了与 BehaviorTree.CPP 的执行路径一致性。

## C++ 参考实现

C++ 参考实现位于 [third_party/BehaviorTree.CPP/](third_party/BehaviorTree.CPP/)。
