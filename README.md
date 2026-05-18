# BehaviorTree — 行为树库 (Go)

[BehaviorTree.CPP](https://github.com/BehaviorTree/BehaviorTree.CPP) 的 Go 移植版本，提供构建模块化、响应式 AI 行为树的框架。

## 包结构

| 包 | 说明 |
|---------|-------------|
| [core](core/) | 核心类型：`TreeNode`、`NodeStatus`、`Blackboard`、`BehaviorTreeFactory`、端口系统 |
| [action](action/) | 内置动作节点：`Script`、`Sleep`、`SetBlackboard`、`AlwaysSuccess/Failure`、`EntryUpdatedAction` 等 |
| [control](control/) | 控制流节点：`Sequence`（及变体）、`Fallback`（及变体）、`Parallel`、`ReactiveSequence/Fallback`、`IfThenElse`、`WhileDoElse`、`TryCatch`、`Switch`（2~5 分支）等 |
| [decorator](decorator/) | 装饰器节点：`Retry`、`Repeat`、`Timeout`、`Delay`、`Inverter`、`ForceSuccess/Failure`、`KeepRunningUntilFailure`、`RunOnce`���`SubTree`、`Precondition`、`Loop`、`UpdatedDecorator` 等 |
| [script](script/) | BT 脚本分词器、解析器和执行器，用于嵌入式条件表达式 |
| [xml](xml/) | 行为树的 XML 序列化和反序列化 |

## 安装

```bash
go get github.com/actfuns/behaviortree
```

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/actfuns/behaviortree/core"
    _ "github.com/actfuns/behaviortree/script"
    _ "github.com/actfuns/behaviortree/xml"
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
- **类型安全**: 端口规则系统验证类型兼容性；字符串作为"通用供体"可自动转换
- **响应式节点**: `ReactiveSequence`、`ReactiveFallback` — 每次 tick 重新评估条件
- **异步支持**: 节点返回 `RUNNING` 表示需要多次 tick；唤醒信号驱动自动重试
- **状态变化监听**: `StateChangeListener` 订阅节点状态变更事件
- **XML 加载**: 完整支持 BehaviorTree.CPP `BTCPP_format="4"` XML 解析，含 `<include>` 和子树
- **脚本表达式**: 支持内联脚本（赋值、算术、比较、三元运算），可用作条件/前置/后置条件
- **三种 tick 模式**: `TickOnce()` / `TickExactlyOnce()` / `TickWhileRunning()`，匹配 C++ API
- **SubTree 支持**: 树组合与端口重映射，子树间黑��板隔离
- **替换规则**: `AddSubstitutionRule` 支持 XML 级别的节点类型替换（通配符匹配）
- **枚举注册**: `RegisterScriptingEnum` 支持在脚本中使用枚举值
- **线程安全**: 工厂和黑板操作支持并发读写
- **测试覆盖**: 405+ 测试用例，涵盖所有核心功能和 C++ 对比测试

## 前置/后置条件

节点支持以下条件属性：

| 属性 | 效果 |
|--------|--------|
| `_successIf="expr"` | 表达式为真时节点返回 SUCCESS |
| `_failureIf="expr"` | 表达式为真时节点返回 FAILURE |
| `_skipIf="expr"` | 表达式为真时节点被跳过 (SKIPPED) |
| `_while="expr"` | 表达式为真时节点可运行，为假时跳过 |
| `_onSuccess="script"` | 节点 SUCCESS 时执行脚本 |
| `_onFailure="script"` | 节点 FAILURE 时执行脚本 |
| `_onHalted="script"` | 节点被中断时执行脚本 |

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
go test github.com/actfuns/behaviortree/...
```

全部 6 个包、405+ 个测试用例通过，覆盖核心模块、控制流节点、装饰器、脚本引擎和 XML 解析。

## C++ 参考实现

C++ 参考实现位于 [third_party/BehaviorTree.CPP/](third_party/BehaviorTree.CPP/)。
