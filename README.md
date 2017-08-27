# goreduce

[![Build Status](https://travis-ci.org/mvdan/goreduce.svg?branch=master)](https://travis-ci.org/mvdan/goreduce)

Reduce a program to its simplest form as long as it produces a compiler
error or any output (such as a panic) matching a regular expression.

	go get -u mvdan.cc/goreduce

### Example

```
func main() {
        a := []int{1, 2, 3}
        if true {
                a = append(a, 4)
        }
        a[1] = -2
        println(a[10])
}
```

	goreduce -match 'index out of range' .

```
func main() {
        a := []int{}
        println(a[0])
}
```

For more usage information, see `goreduce -h`.

### Design

* The tool should be reproducible, giving the same output for an input
  program as long as external factors don't modify its behavior
* The rules should be as simple and composable as possible
* Rules should avoid generating changes that they can know won't compile

### Rules

#### Removing

|                 | Before              | After         |
| --------------- | ------------------- | ------------- |
| statement       | `a; b`              | `a` or `b`    |
| index           | `a[1]`              | `a`           |
| slice           | `a[:2]`             | `a` or `a[:]` |
| binary part     | `a + b`, `a && b`   | `a` or `b`    |
| unary op        | `-a`, `!a`          | `a`           |
| star            | `*a`                | `a`           |
| parentheses     | `(a)`               | `a`           |
| if/else         | `if a { b } else c` | `b` or `c`    |
| defer           | `defer f()`         | `f()`         |
| go              | `go f()`            | `f()`         |
| basic value     | `123, "foo"`        | `0, ""`       |
| composite value | `T{a, b}`           | `T{}`         |

#### Inlining

|                 | Before              | After         |
| --------------- | ------------------- | ------------- |
| const           | `const c = 0; f(c)` | `f(0)`        |
| var             | `v := false; f(v)`  | `f(false)`    |
| case            | `case x: a`         | `a`           |
| block           | `{ a }`             | `a`           |
| simple call     | `f()`               | `{ body }`    |

#### Resolving

|                 | Before              | After         |
| --------------- | ------------------- | ------------- |
| integer op      | `2 * 3`             | `6`           |
| string op       | `"foo" + "bar"`     | `"foobar"`    |
| slice           | `"foo"[1:]`         | `"oo"`        |
| index           | `"foo"[0]`          | `'f'`         |
| builtin         | `len("foo")`        | `3`           |
