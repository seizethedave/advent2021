package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/pkg/errors"
)

const (
	TypeSum = iota
	TypeProduct
	TypeMinimum
	TypeMaximum
	TypeLiteral
	TypeGreaterThan
	TypeLessThan
	TypeEqualTo
)

var ops = [...]func([]int) int{
	TypeSum: func(v []int) int {
		s := 0
		for _, n := range v {
			s += n
		}
		return s
	},
	TypeProduct: func(v []int) int {
		s := 1
		for _, n := range v {
			s *= n
		}
		return s
	},
	TypeMinimum: func(v []int) int {
		s := math.MaxInt
		for _, n := range v {
			if n < s {
				s = n
			}
		}
		return s
	},
	TypeMaximum: func(v []int) int {
		s := 0
		for _, n := range v {
			if n > s {
				s = n
			}
		}
		return s
	},
	TypeGreaterThan: func(v []int) int {
		if v[0] > v[1] {
			return 1
		}
		return 0
	},
	TypeLessThan: func(v []int) int {
		if v[0] < v[1] {
			return 1
		}
		return 0
	},
	TypeEqualTo: func(v []int) int {
		if v[0] == v[1] {
			return 1
		}
		return 0
	},
}

type bitReader struct {
	io.ByteReader
	buf     byte
	balance int
	offset  int
}

func (r *bitReader) readBits(bits int) (int, error) {
	var ret int

	for bits > 0 {
		if r.balance == 0 {
			buf, err := r.ReadByte()
			if err != nil {
				return 0, err
			}
			r.buf = buf
			r.balance = 8
		}

		ret <<= 1
		if (r.buf & (1 << (r.balance - 1))) != 0 {
			ret |= 1
		}

		r.balance--
		bits--
		r.offset++
	}

	return ret, nil
}

func main() {
	bufreader := bufio.NewReader(hex.NewDecoder(os.Stdin))
	bitreader := &bitReader{bufreader, 0, 0, 0}

	sum, err := readPacket(bitreader)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(sum)
}

func readLiteral(r *bitReader) (int, error) {
	val := 0
	cont := true

	for cont {
		word, err := r.readBits(5)
		if err != nil {
			return 0, err
		}
		cont = (word & 0b10000) != 0
		val = (val << 4) | (word & 0b01111)
	}

	return val, nil
}

func readPacket(r *bitReader) (int, error) {
	_, err := r.readBits(3) // version
	if err != nil {
		return 0, err
	}

	packetType, err := r.readBits(3)
	if err != nil {
		return 0, err
	}

	if packetType == TypeLiteral {
		// Eval and return literal.
		lit, err := readLiteral(r)
		if err != nil {
			return 0, errors.Wrap(err, "reading literal")
		}
		return lit, nil
	}

	// Otherwise, a general operator.

	lengthFlag, err := r.readBits(1)
	if err != nil {
		return 0, err
	}

	var subvalues []int

	switch lengthFlag {
	case 1:
		subpackets, err := r.readBits(11)
		if err != nil {
			return 0, err
		}

		for i := 0; i < subpackets; i++ {
			subval, err := readPacket(r)
			if err != nil {
				return 0, errors.Wrap(err, "read subpacket")
			}
			subvalues = append(subvalues, subval)
		}
	case 0:
		length, err := r.readBits(15)
		if err != nil {
			return 0, err
		}

		for begin := r.offset; r.offset < begin+length; {
			subval, err := readPacket(r)
			if err != nil {
				return 0, errors.Wrap(err, "read subpacket")
			}
			subvalues = append(subvalues, subval)
		}
	}

	op := ops[packetType]
	if op == nil {
		return 0, fmt.Errorf("operator %X not found", packetType)
	}
	return op(subvalues), nil
}
