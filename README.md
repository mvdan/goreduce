# goreduce

[![Build Status](https://travis-ci.org/mvdan/goreduce.svg?branch=master)](https://travis-ci.org/mvdan/goreduce)

Reduce a function to its simplest form as long as it produces a compiler
error or any output (such as a panic) matching a regular expression.

Still a work in progress and barely useful.

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

	$ goreduce -match 'index out of range' . Crasher

```
func Crasher() {
        a := []int{1, 2, 3}
        println(a[10])
}
```

### Rules

| Summary             | Before                  | After         |
| ------------------- | ----------------------- | ------------- |
| Remove statement    | `a; b`                  | `a` or `b`    |
| Bypass to if/else   | `if a { b } else { c }` | `b` or `c`    |
| Bypass defer        | `defer a`               | `a`           |
| Zero lit values     | `123, "foo"`            | `0, ""`       |
| Reduce indexes      | `a[1]`                  | `a`           |
| Reduce slices       | `a[:2]`                 | `a` or `a[:]` |
| Remove binary parts | `a + b`, `a || b`       | `a` or `b`    |
| Remove unary op     | `-a`, `!a`              | `a`           |
| Bypass star         | `*a`                    | `a`           |
