---
title: Golang-Array和Slice
date: 2021-10-05 16:14:16
tags:
- Golang
- 编程语言
---
## Spec

> Array:
>
> ```
> ArrayType   = "[" ArrayLength "]" ElementType .
> ArrayLength = Expression .
> ElementType = Type .
> ```
>
> 其中ArrayLength是Array类型的一个组成部分，且必须能够计算为一个非负int类型可“常量表示（[constant](https://golang.org/ref/spec#Constants) [representable](https://golang.org/ref/spec#Representability)）”的值
>
> 
>
> Slice:
>
> ```
> SliceType = "[" "]" ElementType .
> ```
>
> * 一个Slice只要经过了初始化，就必将和一个底层的Array绑定，绑定在同一个Array上的Slices会共享内存
>
> * 通过make初始化一个Slice的同时会初始化一个隐式的Array，也就是说下面两行代码是等价的：
>
>   ```go
>   make([]int, 50, 100)
>   new([100]int)[0:50]
>   ```



## Arrays

Golang的数组类型和C不同：

1. Golang的数组是值，而不像C是数组头指针
2. 所以，在给一个函数传数组参树的时候，实际上是传的整个数组的一份copy
3. 数组长度是类型的一部分，所以`[10]int`和`[20]int`是两个不同的类型



## Slices

内部表示为一个三元素结构体：

1. ptr：指向底层数组的指针
2. len：Slice长度，表示引用的元素个数
3. cap：Slice容量，从Slice引用的第一个元素到底层数组最后一个元素的数量

 ![img](slice-struct.png)

比如`s := make([]int, 5)`，s的表示如下：

 ![img](slice-1.png)

对s进行切片`s = s[2:4]`，s的表示变为下面这样：

 ![img](slice-2.png)

我们可以在cap范围内进行Slice的扩张，比如：`s = s[:cap(s)]`，s的表示变为下面这样：

 ![img](slice-3.png)

超过cap以及访问底层数组中处于Slice更前的元素都是**不被允许的**



#### **扩容**

当向Slice中添加元素的时候发现底层Array的容量已经不够则会触发扩容，底层Array将发生内存重分配。为了减少内存分配操作，我们应该在初始化Slice的时候尽量给出一个预期的cap大小。

扩容逻辑类似与下面这段代码：

```go
func Append(slice, data []byte) []byte {
    l := len(slice)
    if l + len(data) > cap(slice) {  // reallocate
        // Allocate double what's needed, for future growth.
        newSlice := make([]byte, (l+len(data))*2)
        // The copy function is predeclared and works for any slice type.
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0:l+len(data)]
    copy(slice[l:], data)
    return slice
}
```

我们必须将扩容后的Slice返回，因为Slice本身实际上还是值传递（底层字段ptr，len， cap）





#### **潜在的陷阱**

由于Array只有在不被任何地方引用的时候才能够被GC掉，而Slice会隐式地引用一个底层Array，那么会出现下面这种情况：

```go
var digitRegexp = regexp.MustCompile("[0-9]+")

func FindDigits(filename string) []byte {
    b, _ := ioutil.ReadFile(filename)
    return digitRegexp.Find(b)
}
```

* b是一个包含了整个文件内容的Slice

* 返回值是包含了文件内容中第一组连续数字的Slice

* 这两个Slice都引用着存储着所有文件内容的底层Array

当该函数被Caller调用后，实际上真正有用的数据就是返回值Slice所引用的数据。它的底层Array有大量不需要的数据得不到GC。

为了避免这个问题，我们可以将我们需要的数据copy到一个新的Slice中再返回：

```go
func CopyDigits(filename string) []byte {
    b, _ := ioutil.ReadFile(filename)
    b = digitRegexp.Find(b)
    c := make([]byte, len(b))
    copy(c, b)
    return c
}
```



> 本文图片引自：
>
> https://go.dev/blog/slices-intro
