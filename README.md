# goreduce

[![Build Status](https://travis-ci.org/mvdan/goreduce.svg?branch=master)](https://travis-ci.org/mvdan/goreduce)

Reduce a function to its simplest form as long as it produces a compiler
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

	goreduce -match 'index out of range' . Crasher

```
func Crasher() {
        a := []int{}
        println(a[0])
}
```

### Design

* The tool should be reproducible, giving the same output for an input
  program as long as external factors don't modify its behavior
* The rules should be as simple and composable as possible
* Rules should avoid generating changes that they can know won't compile

### Rules

These are tested one at a time. If any of them makes the regular
expression still match, it's left in place.

| Summary              | Before              | After         |
| -------------------- | ------------------- | ------------- |
| Remove statement     | `a; b`              | `a` or `b`    |
| Inline block         | `{ a }`             | `a`           |
| Bypass to if/else    | `if a { b } else c` | `b` or `c`    |
| Bypass to defer call | `defer a()`         | `a()`         |
| Bypass to go call    | `go a()`            | `a()`         |
| Zero lit values      | `123, "foo"`        | `0, ""`       |
| Empty composite lits | `T{a, b}`           | `T{}`         |
| Reduce indexes       | `a[1]`              | `a`           |
| Reduce slices        | `a[:2]`             | `a` or `a[:]` |
| Remove binary parts  | `a + b`, `a || b`   | `a` or `b`    |
| Remove unary op      | `-a`, `!a`          | `a`           |
| Bypass star          | `*a`                | `a`           |
| Bypass paren         | `(a)`               | `a`           |

Note that extra changes may be needed along with any of these to appease
the Go compiler, since it doesn't like unused variables and imports.
