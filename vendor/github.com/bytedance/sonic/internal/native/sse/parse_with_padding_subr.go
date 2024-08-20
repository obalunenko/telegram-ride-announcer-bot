// +build !noasm !appengine
// Code generated by asm2asm, DO NOT EDIT.

package sse

import (
	`github.com/bytedance/sonic/loader`
)

const (
    _entry__parse_with_padding = 336
)

const (
    _stack__parse_with_padding = 192
)

const (
    _size__parse_with_padding = 47736
)

var (
    _pcsp__parse_with_padding = [][2]uint32{
        {0x1, 0},
        {0x6, 8},
        {0x8, 16},
        {0xa, 24},
        {0xc, 32},
        {0xd, 40},
        {0x14, 48},
        {0xc5a, 192},
        {0xc5b, 48},
        {0xc5d, 40},
        {0xc5f, 32},
        {0xc61, 24},
        {0xc63, 16},
        {0xc64, 8},
        {0xc65, 0},
        {0xba78, 192},
    }
)

var _cfunc_parse_with_padding = []loader.CFunc{
    {"_parse_with_padding_entry", 0,  _entry__parse_with_padding, 0, nil},
    {"_parse_with_padding", _entry__parse_with_padding, _size__parse_with_padding, _stack__parse_with_padding, _pcsp__parse_with_padding},
}