# goreduce

Reduce a function to its simplest form as long as it produces a compiler
error or any output (such as a panic) matching an expression.

Still a work in progress and barely useful.

### Example

```
func Reduce() bool {
	a := []int{1, 2, 3}
	if true {
		a = append(a, 4)
	}
	println(a[10])
}
```

	$ goreduce -match 'index out of range' Reduce

```
func Reduce() bool {
	a := []int{1, 2, 3}
	println(a[10])
}
```
