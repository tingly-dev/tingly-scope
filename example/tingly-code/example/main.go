package main

import (
	"fmt"
	"sync"
	"time"

	"example/tingly-code"
)

func main() {
	fmt.Println("=== 线程安全的泛型栈示例 ===\n")

	// 示例 1: 基本操作
	fmt.Println("1. 基本操作示例:")
	basicExample()

	// 示例 2: 不同类型
	fmt.Println("\n2. 不同类型示例:")
	differentTypesExample()

	// 示例 3: 并发安全
	fmt.Println("\n3. 并发安全示例:")
	concurrentExample()

	// 示例 4: 实际应用 - 括号匹配
	fmt.Println("\n4. 实际应用 - 括号匹配:")
	bracketMatchingExample()
}

func basicExample() {
	s := stack.New[int]()

	fmt.Println("  压入元素: 1, 2, 3")
	s.Push(1)
	s.Push(2)
	s.Push(3)

	fmt.Printf("  栈大小: %d\n", s.Size())
	fmt.Printf("  是否为空: %v\n", s.IsEmpty())

	if top, ok := s.Peek(); ok {
		fmt.Printf("  栈顶元素 (Peek): %d\n", top)
	}

	fmt.Println("  弹出所有元素:")
	for !s.IsEmpty() {
		if item, ok := s.Pop(); ok {
			fmt.Printf("    弹出: %d\n", item)
		}
	}

	fmt.Printf("  弹出后是否为空: %v\n", s.IsEmpty())
}

func differentTypesExample() {
	// 字符串栈
	fmt.Println("  字符串栈:")
	strStack := stack.New[string]()
	strStack.Push("Hello")
	strStack.Push("World")
	str, _ := strStack.Pop()
	fmt.Printf("    弹出: %s\n", str)

	// 自定义类型栈
	type Person struct {
		Name string
		Age  int
	}
	fmt.Println("  自定义类型栈:")
	personStack := stack.New[Person]()
	personStack.Push(Person{Name: "Alice", Age: 30})
	personStack.Push(Person{Name: "Bob", Age: 25})
	p, _ := personStack.Pop()
	fmt.Printf("    弹出: %+v\n", p)

	// 指针类型栈
	fmt.Println("  指针类型栈:")
	ptrStack := stack.New[*int]()
	a, b := 42, 100
	ptrStack.Push(&a)
	ptrStack.Push(&b)
	ptr, _ := ptrStack.Pop()
	fmt.Printf("    弹出指针指向的值: %d\n", *ptr)
}

func concurrentExample() {
	s := stack.New[int]()
	numGoroutines := 100
	pushesPerGoroutine := 100

	var wg sync.WaitGroup
	start := time.Now()

	// 并发压入
	fmt.Printf("  启动 %d 个 goroutine，每个压入 %d 个元素...\n",
		numGoroutines, pushesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < pushesPerGoroutine; j++ {
				s.Push(id*pushesPerGoroutine + j)
			}
		}(i)
	}
	wg.Wait()

	pushTime := time.Since(start)
	fmt.Printf("  压入完成，耗时: %v\n", pushTime)
	fmt.Printf("  栈大小: %d (期望: %d)\n", s.Size(), numGoroutines*pushesPerGoroutine)

	// 并发弹出
	start = time.Now()
	fmt.Printf("  启动 %d 个 goroutine 并发弹出...\n", numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < pushesPerGoroutine; j++ {
				s.Pop()
			}
		}()
	}
	wg.Wait()

	popTime := time.Since(start)
	fmt.Printf("  弹出完成，耗时: %v\n", popTime)
	fmt.Printf("  最终栈大小: %d (期望: 0)\n", s.Size())
	fmt.Printf("  总耗时: %v\n", pushTime+popTime)
}

func bracketMatchingExample() {
	// 使用栈检查括号是否匹配
	isValid := func(s string) bool {
		st := stack.New[rune]()
		matching := map[rune]rune{')': '(', '}': '{', ']': '['}

		for _, ch := range s {
			switch ch {
			case '(', '{', '[':
				st.Push(ch)
			case ')', '}', ']':
				if top, ok := st.Pop(); !ok || top != matching[ch] {
					return false
				}
			}
		}
		return st.IsEmpty()
	}

	testCases := []struct {
		expr string
		desc string
	}{
		{"()", "简单匹配"},
		{"({[]})", "嵌套匹配"},
		{"({[)]}", "不匹配"},
		{"((()))", "多层嵌套"},
		{"{[()()]}", "复杂嵌套"},
	}

	fmt.Println("  括号匹配检测结果:")
	for _, tc := range testCases {
		result := "✓ 匹配"
		if !isValid(tc.expr) {
			result = "✗ 不匹配"
		}
		fmt.Printf("    %s: %s -> %s\n", tc.desc, tc.expr, result)
	}
}
