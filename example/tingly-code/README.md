# 线程安全的泛型栈实现 (Thread-Safe Generic Stack in Go)

一个高性能、线程安全的泛型栈实现，支持完整的并发访问。

## 特性

- ✅ **泛型支持** - 使用 Go 1.18+ 泛型，支持任意类型
- ✅ **线程安全** - 使用 `sync.RWMutex` 保证并发安全
- ✅ **完整操作** - Push、Pop、Peek、IsEmpty、Size、Clear、ToSlice
- ✅ **100% 测试覆盖** - 包含边界情况和并发测试
- ✅ **高性能** - 读写锁优化，支持高并发场景

## 安装

```bash
go get example/tingly-code
```

## 使用示例

### 基本使用

```go
package main

import (
    "fmt"
    "example/tingly-code/stack"
)

func main() {
    // 创建一个整数栈
    s := stack.New[int]()
    
    // 压入元素
    s.Push(1)
    s.Push(2)
    s.Push(3)
    
    // 查看栈顶元素
    if top, ok := s.Peek(); ok {
        fmt.Println("栈顶元素:", top) // 输出: 3
    }
    
    // 弹出元素
    if item, ok := s.Pop(); ok {
        fmt.Println("弹出:", item) // 输出: 3
    }
    
    // 获取栈大小
    fmt.Println("栈大小:", s.Size()) // 输出: 2
    
    // 检查是否为空
    fmt.Println("是否为空:", s.IsEmpty()) // 输出: false
}
```

### 使用不同类型

```go
// 字符串栈
strStack := stack.New[string]()
strStack.Push("Hello")
strStack.Push("World")

// 自定义类型栈
type Person struct {
    Name string
    Age  int
}
personStack := stack.New[Person]()
personStack.Push(Person{Name: "Alice", Age: 30})

// 指针类型栈
ptrStack := stack.New[*int]()
val := 42
ptrStack.Push(&val)
```

### 并发安全示例

```go
s := stack.New[int]()
var wg sync.WaitGroup

// 并发压入
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(val int) {
        defer wg.Done()
        s.Push(val)
    }(i)
}
wg.Wait()

fmt.Println("栈大小:", s.Size()) // 输出: 100
```

## API 文档

### `New[T any]() *Stack[T]`
创建一个新的空栈。

### `Push(item T)`
将元素压入栈顶。

### `Pop() (T, bool)`
从栈顶弹出元素。返回元素和是否成功。如果栈为空，返回零值和 false。

### `Peek() (T, bool)`
查看栈顶元素但不移除。返回元素和是否成功。如果栈为空，返回零值和 false。

### `IsEmpty() bool`
检查栈是否为空。

### `Size() int`
返回栈中元素的数量。

### `Clear()`
清空栈中的所有元素。

### `ToSlice() []T`
将栈转换为切片（从底到顶的顺序）。

## 运行测试

```bash
# 运行所有测试（包括竞态检测）
go test -v -race -cover

# 运行性能基准测试
go test -bench=. -benchmem

# 运行压力测试
go test -v -race
```

## 测试覆盖

测试覆盖以下场景：

- ✅ 空栈操作
- ✅ 单元素栈
- ✅ 多元素栈和 LIFO 顺序
- ✅ 泛型类型支持（int, string, 自定义类型, 指针）
- ✅ 并发压入操作
- ✅ 并发弹出操作
- ✅ 并发混合操作（Push/Pop/Peek/Size）
- ✅ 竞态条件检测
- ✅ 压力测试（50 goroutines × 10000 operations）

## 性能基准

在典型硬件上的性能表现：

```
BenchmarkPush-8              10000000               120 ns/op
BenchmarkPop-8               10000000               125 ns/op
BenchmarkPeek-8              50000000                35.2 ns/op
BenchmarkConcurrentPush-8     2000000               650 ns/op
BenchmarkConcurrentPop-8      3000000               580 ns/op
```

## 线程安全保证

- 所有操作都使用适当的锁保护
- 读操作（Peek、IsEmpty、Size、ToSlice）使用读锁，支持并发读取
- 写操作（Push、Pop、Clear）使用写锁，保证互斥访问
- 通过 `-race` 检测，无竞态条件

## License

MIT License
