---
title: Scala隐式转换的理解
tags: 
- Scala
- Language
---

### 将隐式解析看作是方法调用

- 隐式参数

    ImplicitValue: Unit => RequiredType
    
- 隐式转换
    
    ImplicitValue: GivenType => RequiredType.
    
- 在方法没有被定义的类型上调用方法
    
    ImplicitValue: GivenType => ???
    

### 隐式范围（implicit scope）定义

1. Current scope
    - Local scope
    - Current Scope defined by Imports (Explicit Imports and Wildcard Imports)
    
    简单的说就是和变量、标识符等的搜索scope是一样的
    
2. Associated Type
   
   * Function0[RequiredType]
   * Function1[GivenType, RequireType]
   * Function1[GivenType, ???]  
  
    隐式范围包括上述涉及到的类型的伴生对象，如果说上述类型是类型构造器，比如说RequiredType[T]，那么T的隐式范围同样会被搜索。GivenType, RequireType的父类/trait（如果有）的伴生对象同样也会被搜索。


参考：
* https://www.geekabyte.io/2017/12/implicit-scope-and-implicit-resolution.html