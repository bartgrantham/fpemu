package hc55516

// sox -traw -b16 -esigned-integer -r12000 -L out.Dxxx -d

import (
    "fmt"
)

type CVSD struct {
    Shift  uint8
    Filter  float32
    State  float32
}

const FILTER_MIN float32 = -0.08
const FILTER_MAX float32 = 0.08
const FILTER_LEAK float32 = 0.3
const LEAK float32 = 0.1

// *not* the actual CVSD algorithm, produces smoother curves
// may need more careful reconstruction if this ends up too scratchy
// assume this includes the clock
func (c *CVSD) Addbit(bit bool) {
    c.Shift <<= 1
    if bit {
        c.Shift |= 0x01
    }
    if c.Shift & 0x07 == 0x07 {
        c.Filter += (FILTER_MAX - c.Filter) / 2
    }
    if c.Shift & 0x07 == 0x00 {
        c.Filter += (FILTER_MIN - c.Filter) / 2
    }
    c.State += c.Filter
    c.Filter *= FILTER_LEAK

    if c.State < -1 {
        c.State = -1
    }
    if c.State > 1 {
        c.State = 1
    }

    c.State -= c.State * LEAK
}

/*
// the actual algorithm, if memory serves...
func (c *CVSD) Addbit(bit bool) {
    c.Shift <<= 1
    if bit {
        c.Shift |= 0x01
    }
    if c.Shift & 0x07 == 0x07 {
        c.Filter += (FILTER_MAX - c.Filter) * .5
    }

    if c.Filter < FILTER_MIN {
        c.Filter = FILTER_MIN
    }
    if c.Filter > FILTER_MAX {
        c.Filter = FILTER_MAX
    }

    if bit {
        c.State += c.Filter
    } else {
        c.State -= c.Filter
    }
    c.Filter *= FILTER_LEAK

    if c.State < 0 {
        c.State = 0
    }
    if c.State > 1 {
        c.State = 1
    }

    c.State -= (c.State - .5) * LEAK
}
*/

func (c *CVSD) String() string {
    str := ""
    if c.Shift & 0x04 == 0x04 { str += "1 " } else { str += "0 " }
    if c.Shift & 0x02 == 0x02 { str += "1 " } else { str += "0 " }
    if c.Shift & 0x01 == 0x01 { str += "1 " } else { str += "0 " }
    str += fmt.Sprintf("%.4f %.4f", c.Filter, c.State)
    for i:=0; i<int(c.State*80); i++ {
        str += " "
    }
    str += "*"
    return str
}
