package xml

import (
	"bytes"
	"encoding/xml"
	"sort"

	"github.com/actfuns/behaviortree/core"
)

func nodeTypeToElemName(nt core.NodeType) string {
	switch nt {
	case core.Action:
		return "Action"
	case core.Condition:
		return "Condition"
	case core.Control:
		return "Control"
	case core.Decorator:
		return "Decorator"
	case core.Subtree:
		return "SubTree"
	default:
		return "Undefined"
	}
}

func portDirToElemName(d core.PortDirection) string {
	switch d {
	case core.INPUT:
		return "input_port"
	case core.OUTPUT:
		return "output_port"
	case core.INOUT:
		return "inout_port"
	default:
		return "inout_port"
	}
}

// WriteTreeNodesModelXML generates the <TreeNodesModel> XML for all node
// manifests registered in the factory. When includeBuiltin is false, the
// built-in node types (e.g. SubTree) are excluded.
func WriteTreeNodesModelXML(factory core.BehaviorTreeFactory, includeBuiltin bool) string {
	manifests := factory.Manifests()
	builtins := factory.BuiltinNodes()

	ids := make([]string, 0, len(manifests))
	for id := range manifests {
		if !includeBuiltin && builtins[id] {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	// <root BTCPP_format="4">
	rootStart := xml.StartElement{
		Name: xml.Name{Local: "root"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "BTCPP_format"}, Value: "4"}},
	}
	enc.EncodeToken(rootStart)

	// <TreeNodesModel>
	modelStart := xml.StartElement{Name: xml.Name{Local: "TreeNodesModel"}}
	enc.EncodeToken(modelStart)

	for _, id := range ids {
		manifest := manifests[id]

		elemName := nodeTypeToElemName(manifest.Type)
		nodeStart := xml.StartElement{
			Name: xml.Name{Local: elemName},
			Attr: []xml.Attr{{Name: xml.Name{Local: "ID"}, Value: manifest.RegistrationID}},
		}
		enc.EncodeToken(nodeStart)

		// Sort ports by name for deterministic output
		portNames := make([]string, 0, len(manifest.Ports))
		for name := range manifest.Ports {
			portNames = append(portNames, name)
		}
		sort.Strings(portNames)

		for _, portName := range portNames {
			portInfo := manifest.Ports[portName]
			portElem := portDirToElemName(portInfo.Direction())

			portStart := xml.StartElement{
				Name: xml.Name{Local: portElem},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "name"}, Value: portName},
				},
			}

			typeName := portInfo.TypeName()
			if typeName != "" && typeName != "AnyTypeAllowed" {
				portStart.Attr = append(portStart.Attr,
					xml.Attr{Name: xml.Name{Local: "type"}, Value: typeName})
			}

			if def := portInfo.DefaultValueString(); def != "" {
				portStart.Attr = append(portStart.Attr,
					xml.Attr{Name: xml.Name{Local: "default"}, Value: def})
			}

			enc.EncodeToken(portStart)
			if desc := portInfo.Description(); desc != "" {
				enc.EncodeToken(xml.CharData(desc))
			}
			enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: portElem}})
		}

		if len(manifest.Metadata) > 0 {
			metaFieldsStart := xml.StartElement{Name: xml.Name{Local: "MetadataFields"}}
			enc.EncodeToken(metaFieldsStart)
			for _, kv := range manifest.Metadata {
				metaStart := xml.StartElement{
					Name: xml.Name{Local: "Metadata"},
					Attr: []xml.Attr{{Name: xml.Name{Local: kv.Key}, Value: kv.Value}},
				}
				enc.EncodeToken(metaStart)
				enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "Metadata"}})
			}
			enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "MetadataFields"}})
		}

		enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: elemName}})
	}

	enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "TreeNodesModel"}})
	enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "root"}})

	enc.Flush()
	return buf.String()
}
