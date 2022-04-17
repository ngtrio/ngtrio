---
title: Go泛型初体验
date: 2022-04-17 23:09:41
tags:
- Go
- 编程语言
---
![](2022-04-17-15-22-16.png)
* Generics
* Fuzzing
* Workspaces
* 20% Performance Improvements

本文是对Go1.18的第一个特性：泛型（Generics）做的一些体验总结。

## 从常见的场景说起

### 对指针类型进行解引用  

一般我们会像下面这样对不同的类型分别实现对应的解引用函数（不使用反射的情况下）
```go
func PtrToString(strPtr *string) string {
    if strPtr == nil {
        return ""
    }

    return *strPtr
}

func PtrToInt(intPtr *int) int {
    if intPtr == nil {
        return 0
    }

    return *intPtr
}

func PtrToBool(boolPtr *bool) bool {
    if boolPtr == nil {
        return false
    }

    return *boolPtr
}

// ...
```
每新增一个类型我们就需要针对这个类型实现一段类似重复的代码，但这些函数的唯一不同点，**实际上只有函数参数的类型和函数响应值的类型不同**。   

而 **泛型(Generics)** 就是给变量的类型也引入“形参&实参”的概念。我们定义函数的时候可以给参数类型定义“类型形参”，在调用函数的时候，我们可以给函数传入类型的实参。如果我们使用泛型，上述的代码实现可以转变为下面这样：   
```go
// 定义解引用函数
func FromPtr[T any](anyPtr *T) T {
	var zero T
	if anyPtr == nil {
		return zero
	}
	return *anyPtr
}
```
对于不同的类型，我们都只需要调用这一个`泛型函数`即可。   
```go
// 调用解引用函数
func main() {
	var intPtr *int
	var strPtr *string
	var boolPtr *bool
	a := 1
	b := "str"
	c := true

    // 类型实参自动推断，所以不需要我们像下面这样手动传入了
    // fmt.Println(FromPtr[int](intPtr))
	fmt.Println(FromPtr(intPtr))  // 0
	fmt.Println(FromPtr(strPtr))  // ""
	fmt.Println(FromPtr(boolPtr)) // false
	fmt.Println(FromPtr(&a))      // 1
	fmt.Println(FromPtr(&b))      // str
	fmt.Println(FromPtr(&c))      // true
}
```

### 自定义容器
假设我们想定义一个能够做元素累加的容器   
以往我们需要为每个基础类型分别定义一个容器，并相应实现对应的累加方法：   
```go
type IntSlice []int
type StrSlice []string

func(s IntSlice) AddAll() int {/*省略*/}
func(s StrSlice) AddAll() string {/*省略*/}
```
当有了泛型特性之后，我们只需要定义一个`泛型类型`, 然后实现累加的`泛型方法`即可:   
```go
type AnySlice[T int | string] []T
func (s AnySlice[T]) AddAll() T {
	var ret T
	for _, elem := range s {
		ret += elem
	}
	return ret
}
```
我们可以这样使用定义好的`泛型类型`
```go
intSlice := AnySlice[int]{1, 2, 3, 4, 5}
strSlice := AnySlice[string]{"str1", "str2", "str3", "str4", "str5"}

fmt.Println(intSlice.AddAll()) // 15
fmt.Println(strSlice.AddAll()) // str1str2str3str4str5
```

## 基本概念
我们使用上面定义的泛型函数引入一些基本概念   
``` go
    类型参数列表   类型约束
            |    |
func FromPtr[T any](anyPtr *T) T   - 泛型函数
              \            /_/
                \ 类型形参 /   

   类型实参
         |
FromPtr[int](intPtr)               - 泛型函数调用

    类型参数列表     类型约束
             |      |
             |  ------------
type AnySlice[T int | string] []T  - 泛型类型
               \               /
                 \  类型形参   /   

      泛型接收器
      |
func (s AnySlice[T]) AddAll() T    - 泛型方法

    类型实参
         |
AnySlice[int]{1, 2, 3, 4, 5}       - 泛型类型实例化

```

Go泛型的大部分基本概念其实和其他拥有泛型特性的语言基本类似，就不再过多赘述，下面针对不太一样的类型约束做一些介绍。

## 类型约束(type constraint)
了解类型约束的机制，我们才能明白什么样的类型才能实例化我们定义的泛型。   
Go语言本身的类型系统比较简单，不像Scala、Kotlin等语言，有诸如协变、逆变等概念，Go的类型是「不变的」，因此下面的代码会在编译期就报错
```go
type Container[T A] []T

func Append(c Container[any]) Container[any] {
	return append(c, "abc")
}
a := make(Container[int], 0)
Append(a) // 编译期报错，cannot use a (variable of type Container[int]) as Container[any] value in argument to Append compilerIncompatibleAssign
```
Go泛型的类型是通过「类型集」来约束的。下面类型形参后面的部分就是类型约束：
```go
func FromPtr[T any](anyPtr *T) T    // T 约束为 any, 等同于接受所有类型
type AnySlice[T int | string] []T   // T 只接受 int, string
```
上面的例子中「any」是个类型集，包含了所有的类型。「int | string」也是个类型集，包含了int和string两个类型。  

另外为了使得代码更加容易维护，我们可以通过interface来定义类型集，比如下面这个interface代表所有int类型的类型集：
```go
type Int interface {
    int | int8 | int16 | int32 | int64
}

// 然后利用Int来做约束
type GType[T Int] []T
```
上面的例子中，「Int」接口是int、int8、int16、int32、int64的一个类型集，那么这五个类型都可以去实例化泛型类型「GType」。 

实际上，Go的Specification中已经指明interface定义了一个类型集：`An interface type defines a type set`

我们知道Go的interface在之前都只能定义接口方法，本次引入泛型后还能定义一组类型。为了保证向前兼容，interface被分成了下面两种。
### 基础接口（Basic interfaces）
只包含方法的interface。   
假设类型`T`实现了基本接口`I`中定义的全部方法，那么我们称类型`T`实现了接口`I`, 类型`T`满足接口`I`的类型约束。(有点类似于其他语言中的类型上界的味道)   
举例：
```go
type Type[T Parent] []T  // 泛型类型，类型约束为Parent

type Parent interface {
	Func()
}

type Child struct {}
func (c Child) Func() {}

type Child1 struct {}

var t Type[Child]  // ok, 因为Child实现了Parent
var t1 Type[Child1] // wrong, Child1没有实现方法Func
```

### 一般接口（General interfaces）
除了包含方法，还会包含`t1|t2|…|tn`这种形式的类型定义的interface。   
假设类型`T`是接口`I`定义的`t1|t2|…|tn`中的一个，并且实现了接口`I`的所有方法。那么我们称类型`T`实现了接口`I`, 类型`T`满足接口`I`的类型约束。   
举例：
```go
type Type[T Parent] []T  // 泛型类型，类型约束为Parent

type Parent interface {
	Child | Child1
	Func()
}

type Child struct{}
func (c Child) Func() {}

type Child1 struct{}

type Child2 struct{}
func (c Child2) Func() {}

var t Type[Child]  // ok, 因为Child实现了Parent
var t1 Type[Child1] // wrong, Child1没有实现方法Func
var t2 Type[Child2] // wrong, Child2实现了方法Func，但不是 Child | Child1 中的一员
```

## 后续
* 接口交集、并集
* 底层类型约束(~)
* 泛型接口