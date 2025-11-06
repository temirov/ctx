package output

import (
	"github.com/tyemirov/ctx/internal/types"
)

func RenderCallChainToon(data *types.CallChainOutput) string {
	if data == nil {
		return ""
	}
	builder := toonBuilder{}
	builder.ensureLineBreak()
	builder.writeIndent(0)
	builder.buffer.WriteString("callchain:\n")
	builder.writeField(1, toonField{key: "targetFunction", value: data.TargetFunction})
	builder.writeScalarArray(1, "callers", data.Callers)
	if data.Callees != nil {
		builder.writeScalarArray(1, "callees", *data.Callees)
	}
	names := orderedFunctionNames(data)
	builder.writeArrayHeader(1, "functions", len(names))
	for _, name := range names {
		fields := []toonField{
			{key: "name", value: name},
			{key: "body", value: data.Functions[name]},
		}
		builder.writeObjectInArray(2, fields)
	}
	if len(data.Documentation) > 0 {
		builder.writeArrayHeader(1, "documentation", len(data.Documentation))
		for _, entry := range data.Documentation {
			builder.writeDocumentationEntry(2, entry)
		}
	}
	return builder.String()
}
