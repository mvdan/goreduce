# goreduce

[![Build Status](https://travis-ci.org/mvdan/goreduce.svg?branch=master)](https://travis-ci.org/mvdan/goreduce)

Reduce a program to its simplest form as long as it produces a compiler
error or any output (such as a panic) matching a regular expression.

	go get -u github.com/mvdan/goreduce

### Example

```
func Crasher() {
        a := []int{1, 2, 3}
        if true {
                a = append(a, 4)
        }
        a[1] = -2
        println(a[10])
}
```

	goreduce -match 'index out of range' -call Crasher .

```
func Crasher() {
        a := []int{}
        println(a[0])
}
```

And for compiler crashes:

	goreduce -match 'internal compiler error' . -gcflags '-c=2'

### Design

* The tool should be reproducible, giving the same output for an input
  program as long as external factors don't modify its behavior
* The rules should be as simple and composable as possible
* Rules should avoid generating changes that they can know won't compile

### Rules

These are changes made to the AST - single steps towards reducing a
program. They go from removing a statement to inling a variable and
anything in between. See `rules.go`.
