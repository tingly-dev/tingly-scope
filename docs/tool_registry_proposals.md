# 工具注册简化方案

## 问题分析

当前工具注册流程过于复杂：

```go
// 1. 创建 TypedToolkit
tt := tools.NewTypedToolkit()

// 2. 创建工具实例
fileTools := tools.NewFileTools(workDir)

// 3. 调用单独的注册函数
registerTypedFileTools(tt, fileTools)

// 4. 需要为每个工具创建 wrapper 类型
//    ViewFileTool, ReplaceFileTool, EditFileTool, ...
// 5. 每个 wrapper 需要实现 Tool 接口
//    Name(), Description(), ParameterSchema(), Call()
```

---

## 方案 A：结构体标签 + 自动注册（推荐）

### 定义工具

```go
// tools/file_tools_auto.go

// 方法级别：tool 标签定义工具名称和描述
// 参数级别：description 标签定义每个参数的说明

type ViewFileParams struct {
    Path   string `json:"path" required:"true" description:"Path to the file to read"`
    Limit  int    `json:"limit,omitempty" description:"Maximum number of lines to return (0 = all)"`
    Offset int    `json:"offset,omitempty" description:"Line number to start reading from (0-based)"`
}

//tool name="view_file" description="Read file contents with line numbers"
func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
    fullPath := filepath.Join(ft.workDir, params.Path)
    content, err := os.ReadFile(fullPath)
    // ...
    return string(content), nil
}

type ReplaceFileParams struct {
    Path    string `json:"path" required:"true" description:"Path to the file to create/overwrite"`
    Content string `json:"content" required:"true" description="Content to write to the file"`
}

//tool name="replace_file" description="Create or overwrite a file with content"
func (ft *FileTools) ReplaceFile(ctx context.Context, params ReplaceFileParams) (string, error) {
    // ...
}

type EditFileParams struct {
    OldText string `json:"old_text" required:"true" description:"Exact text to replace (must match exactly)"`
    NewText string `json:"new_text" required:"true" description="New text to insert"`
    Path    string `json:"path,omitempty" description:"File path (default: use work directory)"`
}

//tool name="edit_file" description="Replace specific text in a file"
func (ft *FileTools) EditFile(ctx context.Context, params EditFileParams) (string, error) {
    // ...
}
```

### 注册工具

```go
// agent/tingly_agent.go

func CreateTinglyAgent(cfg *config.AgentConfig, workDir string) (*agent.ReActAgent, error) {
    // ...

    // 一行注册所有工具
    tt := tools.NewTypedToolkit()
    tt.RegisterAll(tools.NewFileTools(workDir))
    tt.RegisterAll(tools.NewBashTools(bashSession))
    tt.RegisterAll(tools.NewNotebookTools(workDir))

    // ...
}
```

### 实现原理

```go
// tools/registry.go

import (
    "context"
    "reflect"
    "strings"
)

// RegisterAll 自动注册结构体的所有工具方法
func (tt *TypedToolkit) RegisterAll(provider any) error {
    val := reflect.ValueOf(provider)
    typ := val.Type()

    for i := 0; i < typ.NumMethod(); i++ {
        method := typ.Method(i)

        // 检查是否有 tool 标签
        toolTag := method.Tag.Get("tool")
        if toolTag == "" {
            continue
        }

        // 解析标签：name="xxx" description="xxx"
        name := parseTag(toolTag, "name")
        if name == "" {
            name = toSnakeCase(method.Name)
        }
        description := parseTag(toolTag, "description")

        // 创建反射工具包装器
        tool := &ReflectTool{
            name:        name,
            description: description,
            method:      method,
            receiver:    provider,
        }

        // 推断参数类型
        if method.Type.NumIn() == 3 {
            paramType := method.Type.In(2)
            tool.paramType = paramType
            tool.paramSchema = StructToSchema(reflect.New(paramType).Elem().Interface())
        }

        tt.Register(tool)
    }

    return nil
}

// ReflectTool 反射工具包装器
type ReflectTool struct {
    name        string
    description string
    method      reflect.Method
    receiver    any
    paramType   reflect.Type
    paramSchema map[string]any
}

func (rt *ReflectTool) Name() string        { return rt.name }
func (rt *ReflectTool) Description() string { return rt.description }
func (rt *ReflectTool) ParameterSchema() map[string]any {
    return rt.paramSchema
}

func (rt *ReflectTool) Call(ctx context.Context, params any) (string, error) {
    // 将 params 转换为结构体
    paramValues := reflect.New(rt.paramType)
    if err := MapToStruct(params.(map[string]any), paramValues.Interface()); err != nil {
        return "", err
    }

    // 反射调用方法
    results := rt.method.Func.Call([]reflect.Value{
        reflect.ValueOf(rt.receiver),
        reflect.ValueOf(ctx),
        paramValues,
    })

    // 返回结果
    if results[1].IsNil() {
        return results[0].String(), nil
    }
    return "", results[1].Interface().(error)
}
```

### 生成的 JSON Schema 示例

基于上面的标签定义，`StructToSchema` 会生成如下 JSON Schema：

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Path to the file to read"
    },
    "limit": {
      "type": "integer",
      "description": "Maximum number of lines to return (0 = all)"
    },
    "offset": {
      "type": "integer",
      "description": "Line number to start reading from (0-based)"
    }
  },
  "required": ["path"]
}
```

### 标签层次总结

| 位置 | 标签 | 作用 | 示例 |
|------|------|------|------|
| **方法** | `//tool` | 标记为工具 | `//tool name="view_file" description="..."` |
| **结构体字段** | `json` | 参数名和类型 | `json:"path,omitempty"` |
| **结构体字段** | `description` | 参数说明 | `description:"Path to the file"` |
| **结构体字段** | `required` | 是否必填 | `required:"true"` |



### 优点
- ✅ 最简洁：只需加标签
- ✅ 类型安全：编译期检查
- ✅ 无需 wrapper 类型
- ✅ 零冗余代码

### 缺点
- ⚠️ 使用反射（但仅在注册时）

---

## 方案 B：Builder 模式

### 定义工具

```go
// 工具定义不变，仍需要 wrapper 类型
type ViewFileTool struct {
    ft *FileTools
}

func (t *ViewFileTool) Call(ctx context.Context, params any) (string, error) {
    // ...
}
```

### 注册工具

```go
tt := tools.NewBuilder().
    Add(tools.NewFileTools(workDir)).
    Add(tools.NewBashTools(session)).
    Add(tools.NewNotebookTools(workDir)).
    Build()
```

### 优点
- ✅ 链式调用，可读性好
- ✅ 无需反射

### 缺点
- ❌ 仍需 wrapper 类型
- ❌ 代码量减少有限

---

## 方案 C：代码生成（go generate）

### 定义工具

```go
//go:generate toolgen -type FileTools

type FileTools struct {
    workDir string
}

// ViewFile reads file contents
func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
    // 实现
}
```

### 生成代码

```bash
go generate ./...
```

### 优点
- ✅ 零反射开销
- ✅ 编译期类型安全

### 缺点
- ❌ 增加 build 步骤
- ❌ 需要额外代码生成工具
- ❌ 复杂度高

---

## 方案 D：函数式注册

### 定义工具

```go
type FileTools struct {
    workDir string
}

func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
    // 实现
}
```

### 注册工具

```go
tt := tools.NewTypedToolkit()

// 使用辅助函数注册
tt.RegisterFunc(
    tools.NewFileTools(workDir),
    (*FileTools).ViewFile,
    tools.ToolSpec{
        Name:        "view_file",
        Description: "Read file with line numbers",
    },
)
```

### 优点
- ✅ 无需 wrapper 类型
- ✅ 类型安全

### 缺点
- ❌ 每个工具仍需一行注册代码
- ❌ 重复的 ToolSpec 定义

---

## 推荐：方案 A

| 维度 | 方案 A | 方案 B | 方案 C | 方案 D |
|------|--------|--------|--------|--------|
| 简洁性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 类型安全 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 性能 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 易用性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

**方案 A 最佳平衡点**：反射开销仅在注册时，运行时无影响，代码最简洁。
