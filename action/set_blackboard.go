package action

import (
	"log/slog"
	"reflect"

	"github.com/actfuns/behaviortree/core"
)

// SetBlackboardNode stores a string value into an entry of the Blackboard
// specified in "output_key".
type SetBlackboardNode struct {
	core.SyncActionNode
}

func NewSetBlackboardNode(name string, config core.NodeConfig) *SetBlackboardNode {
	n := &SetBlackboardNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("SetBlackboard")
	return n
}

func (n *SetBlackboardNode) Tick() core.NodeStatus {
	var outputKey string
	if err := n.GetInput("output_key", &outputKey); err != nil {
		slog.Error("missing port [output_key]")
		return core.FAILURE
	}

	valueStr := n.GetRawPortValue("value")

	dstEntry := n.Config().Blackboard.GetEntry(outputKey)

	var outValue core.Any

	if ok, strippedKey := core.IsBlackboardPointer(valueStr); ok {
		inputKey := strippedKey
		srcEntry := n.Config().Blackboard.GetEntry(inputKey)
		if srcEntry == nil {
			slog.Error("Can't find the port referred by [value]")
			return core.FAILURE
		}
		if dstEntry == nil {
			if _, err := n.Config().Blackboard.CreateEntry(outputKey, core.NewPortInfoTyped(core.INOUT, srcEntry.Info())); err != nil {
				slog.Error("SetBlackboard failed", "error", err)
				return core.FAILURE
			}
			dstEntry = n.Config().Blackboard.GetEntry(outputKey)
		}

		locked := srcEntry.GetValue()
		if locked != nil {
			outValue = *locked
		} else {
			return core.FAILURE
		}
	} else {
		outValue = core.AnyOf(valueStr)
	}

	if outValue.IsEmpty() {
		return core.FAILURE
	}

	// avoid type issues when port is remapped
	if dstEntry != nil {
		dstInfo := dstEntry.Info()
		if dstInfo.Type() != nil && dstInfo.Type().Kind() != reflect.String && outValue.IsString() {
			if s, err := outValue.ToString(); err == nil {
				var convErr error
				outValue, convErr = dstInfo.ParseString(s)
				if convErr != nil || outValue.IsEmpty() {
					slog.Error("Can't convert string to type: conversion failed",
						s, dstInfo.TypeName())
				}
			}
		}
	}

	if err := n.Config().Blackboard.Set(outputKey, outValue); err != nil {
		slog.Error("SetBlackboard failed", "error", err)
		return core.FAILURE
	}
	return core.SUCCESS
}
