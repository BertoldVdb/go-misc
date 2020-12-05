package pdu

import (
	"encoding/hex"
)

type PDU struct {
	buf []byte

	leftIndex      int
	intiailLeftCap int

	state int
}

func Alloc(headerCap int, dataLen int, dataCap int) *PDU {
	assert(dataLen <= dataCap, "More data requested than capacity")

	buf := make([]byte, headerCap+dataLen, headerCap+dataCap)

	return &PDU{
		buf:            buf,
		leftIndex:      headerCap,
		intiailLeftCap: headerCap,
	}
}

func (p *PDU) SetState(expected int, new int) {
	assert(p.state == expected, "PDU state is not as expected")
	p.state = new
}

func (p *PDU) reallocInternal(leftCap int, dataLen int, dataCap int, copyData bool) bool {
	assert(leftCap >= 0, "Left capacity was < 0")
	assert(dataCap >= 0, "Right capacity was < 0")
	assert(dataLen >= 0, "Data length was < 0")
	assert(dataLen <= dataCap, "Data length was larger than requested capacity")

	if copyData {
		dataLen = 0
	}

	totalCap := leftCap + dataCap
	if cap(p.buf) < totalCap || (cap(p.buf)*64 >= totalCap && totalCap > 512) {
		newPDU := Alloc(leftCap, dataLen, dataCap)
		newPDU.state = p.state

		if copyData {
			newPDU.Append(p.Buf()...)
		}

		*p = *newPDU
		return true
	}

	buf := p.buf[:cap(p.buf)]
	buf = buf[:leftCap+dataLen]

	newPDU := &PDU{
		buf:            buf,
		leftIndex:      leftCap,
		intiailLeftCap: leftCap,
		state:          p.state,
	}

	if copyData {
		newPDU.Append(p.Buf()...)
	}

	*p = *newPDU

	return false
}

func (p *PDU) Realloc(leftCap int, dataLen int, dataCap int) bool {
	return p.reallocInternal(leftCap, dataLen, dataCap, false)
}

func (p *PDU) NormalizeLeft(leftCap int) {
	newCap := cap(p.buf) - leftCap
	if newCap < 0 {
		newCap = 0
	}
	p.reallocInternal(leftCap, 0, newCap, true)
}

func (p *PDU) Reset() {
	p.buf = p.buf[:p.intiailLeftCap]
	p.leftIndex = p.intiailLeftCap
}

func (p *PDU) Buf() []byte {
	return p.buf[p.leftIndex:]
}

func (p *PDU) Len() int {
	return len(p.buf) - p.leftIndex
}

func (p *PDU) RightCap() int {
	return cap(p.buf) - p.leftIndex
}

func (p *PDU) LeftCap() int {
	return p.leftIndex
}

func (p *PDU) Truncate(length int) {
	assert(length >= 0, "Length was < 0")
	assert(length <= p.RightCap(), "Length was > rightCap")

	p.buf = p.buf[:(p.leftIndex + length)]
}

func (p *PDU) DropLeft(amount int) []byte {
	assert(amount >= 0, "amount was < 0")

	if amount > p.Len() {
		return nil
	}

	old := p.leftIndex
	p.leftIndex += amount

	return p.buf[old:p.leftIndex]
}

func (p *PDU) DropRight(amount int) []byte {
	assert(amount >= 0, "amount was < 0")
	if amount > p.Len() {
		return nil
	}

	oldLen := p.Len()
	newLen := oldLen - amount

	p.buf = p.buf[:p.leftIndex+newLen]
	return p.buf[p.leftIndex+newLen : p.leftIndex+oldLen]
}

func (p *PDU) ExtendLeft(amount int) []byte {
	assert(amount >= 0, "Amount was < 0")

	if amount > p.LeftCap() {
		assert(p.reallocInternal(p.LeftCap()+2*amount, 0, p.RightCap(), true), "Requested realloc that was not needed")
	}

	oldLeftIndex := p.leftIndex
	p.leftIndex -= amount

	return p.buf[p.leftIndex:oldLeftIndex]
}

func (p *PDU) ExtendRight(amount int) []byte {
	assert(amount >= 0, "Amount was < 0")

	if amount > p.RightCap()-p.Len() {
		assert(p.reallocInternal(p.LeftCap(), 0, p.RightCap()+2*amount, true), "Requested realloc that was not needed")
	}

	oldLen := len(p.buf)
	p.buf = p.buf[:len(p.buf)+amount]

	return p.buf[oldLen:]
}

func (p *PDU) Append(b ...byte) {
	p.buf = append(p.buf, b...)
}

func (p PDU) String() string {
	return hex.EncodeToString(p.Buf())
}

func assert(condition bool, reason string) {
	if !condition {
		panic(reason)
	}
}
