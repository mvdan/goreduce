# goreduce

Reduce a function to its simplest form as long as it returns true.

Still a work in progress and barely useful.

### Example

```
func Reduce() bool {
	var a int
	if true {
		a = 3
	}
	return a >= 0
}
```

	$ goreduce Reduce

```
func Reduce() bool {
	var a int
	return a >= 0
}
```
