# artlist-grabber
Grab songs from artlist.io

To build, get a Go toolchain and run
`go get github.com/terorie/artlist-grabber`

Usage:
```
Usage of ./artlist-grabber:
  -conns int
        Concurrency (default 4)
  -download
        Download found songs
  -max int
        Max page (default 9999)
  -min int
        Min page
  -no-csv
        Don't create CSV file
```
