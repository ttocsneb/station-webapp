package util

import "github.com/oxtoacart/bpool"

var BufPool *bpool.BufferPool = bpool.NewBufferPool(64)
