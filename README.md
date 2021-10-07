# 🔍 fq
A jq-like tool that queries files via glob.

## ✅ Prerequisites
- Go 1.17+
- jq (installed and on PATH)

## 📦 Installation
```
$ go get github.com/siketyan/fq
$ go install github.com/siketyan/fq
```

## ✨ Usage
```
$ fq ./**/*.go '.[] | map(.name)'
```

The query at 2nd argument are passed to jq.
This means you can use jq syntax fully with fq.
