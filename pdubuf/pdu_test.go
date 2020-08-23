package pdu

import (
	"math/rand"
	"testing"
)

func check(t *testing.T, condition bool, reason ...interface{}) {
	if !condition {
		t.Error(reason...)
		t.FailNow()
	}
}

func checkCapLen(t *testing.T, p *PDU) {
	check(t, cap(p.Buf()) == p.RightCap(), "cap(Buf()) not rightcap")
	check(t, len(p.Buf()) == p.Len(), "len(Buf()) not Len")
	check(t, p.Len() >= 0, "Len() was negative")
	check(t, p.RightCap() >= 0, "RightCap() was negative")
	check(t, p.LeftCap() >= 0, "LeftCap() was negative")
}

func testExtendLeft(t *testing.T, p *PDU, amount int) {
	value := byte(rand.Int())
	value2 := byte(rand.Int())

	l := p.Len()
	if l > 0 {
		p.Buf()[0] = value
	}
	new := p.ExtendLeft(amount)

	checkCapLen(t, p)
	check(t, l+amount == p.Len(), "testExtendLeft: Amount did not increase by expected amount")
	check(t, l == 0 || p.Buf()[amount] == value, "testExtendLeft: Array did not move correctly")
	check(t, len(new) == amount, "testExtendLeft: Returned extend slice has wrong length")

	if amount > 0 {
		new[0] = value2
		check(t, p.Buf()[0] == value2, "testExtendLeft: Returned extend slice was not at the right positon")
	}
}

func testDropLeft(t *testing.T, p *PDU, amount int) {
	value := byte(rand.Int())
	value2 := byte(rand.Int())

	l := p.Len()
	if l < amount {
		check(t, p.DropLeft(amount) == nil, "Was able to drop more than the length")
		checkCapLen(t, p)

		return
	}
	if l == amount {
		return
	}

	p.Buf()[0] = value2
	p.Buf()[amount] = value
	old := p.DropLeft(amount)

	checkCapLen(t, p)
	check(t, l-amount == p.Len(), "testDropLeft: Amount did not increase by expected amount")
	check(t, p.Buf()[0] == value, "testDropLeft: Array did not move correctly")
	check(t, len(old) == amount, "testDropLeft: Returned extend slice has wrong length")

	if amount > 0 {
		check(t, old[0] == value2, "testDropLeft: Returned drop slice was not at the right positon")
	}
}

func testDropRight(t *testing.T, p *PDU, amount int) {
	value := byte(rand.Int())
	value2 := byte(rand.Int())

	l := p.Len()
	if l < amount {
		check(t, p.DropRight(amount) == nil, "Was able to drop more than the length")
		checkCapLen(t, p)

		return
	}
	if l == amount {
		return
	}

	p.Buf()[l-1] = value2
	p.Buf()[l-1-amount] = value
	old := p.DropRight(amount)

	checkCapLen(t, p)
	check(t, l-amount == p.Len(), "testDropRight: Amount did not increase by expected amount")
	check(t, p.Buf()[p.Len()-1] == value, "testDropRight: Array did not move correctly")
	check(t, len(old) == amount, "testDropRight: Returned extend slice has wrong length")

	if amount > 0 {
		check(t, old[len(old)-1] == value2, "testDropRight: Returned drop slice was not at the right positon")
	}
}

func testExtendRight(t *testing.T, p *PDU, amount int) {
	value := byte(rand.Int())
	value2 := byte(rand.Int())

	l := p.Len()
	if l > 0 {
		p.Buf()[0] = value
	}
	new := p.ExtendRight(amount)

	checkCapLen(t, p)
	check(t, l+amount == p.Len(), "testExtendRight: Amount did not increase by expected amount")
	check(t, l == 0 || p.Buf()[0] == value, "testExtendRight: Array did not move correctly")
	check(t, len(new) == amount, "testExtendRight: Returned extend slice has wrong length")

	if amount > 0 {
		new[0] = value2
		check(t, p.Buf()[l] == value2, "testExtendRight: Returned extend slice was not at the right positon")
	}
}

func testAppend(t *testing.T, p *PDU, amount int) {
	value := byte(rand.Int())
	value2 := byte(rand.Int())

	buf := make([]byte, amount)
	if amount > 0 {
		buf[0] = value2
	}

	l := p.Len()
	if l > 0 {
		p.Buf()[0] = value
	}
	p.Append(buf...)

	checkCapLen(t, p)
	check(t, l+amount == p.Len(), "testAppend: Amount did not increase by expected amount")
	check(t, l == 0 || p.Buf()[0] == value, "testAppend: Array did not move correctly")

	if amount > 0 {
		check(t, p.Buf()[l] == value2, "testAppend: Returned extend slice was not at the right positon")
	}
}

func testReset(t *testing.T, p *PDU, useReset bool, amount int) {
	totalCap := p.LeftCap() + p.RightCap()

	if useReset {
		p.Reset()
		amount = 0
	} else {
		p.Truncate(amount)
	}

	check(t, p.Len() == amount, "testReset: Length was wrong")
	check(t, totalCap == p.LeftCap()+p.RightCap(), "testReset: Capacity changed due to reset")
	checkCapLen(t, p)
}

func testNormalizeReset(t *testing.T, p *PDU, leftCap int) {
	if p.Len() == 0 {
		return
	}

	value := byte(rand.Int())

	oldLen := p.Len()
	p.Buf()[0] = value
	p.NormalizeLeft(leftCap)
	check(t, p.LeftCap() == leftCap, "testNormalizeReset: Left cap was wrong")
	check(t, p.Len() == oldLen, "testNormalizeReset: Length changed")
	check(t, p.Buf()[0] == value, "testNormalizeReset: Value changed")
	checkCapLen(t, p)
}

func testPDU(t *testing.T, p *PDU) {
	for i := 0; i < 6000; i++ {
		switch rand.Int() % 5 {
		case 0:
			testExtendLeft(t, p, rand.Intn(20))
		case 1:
			testExtendRight(t, p, rand.Intn(20))
		case 2:
			testAppend(t, p, rand.Intn(20))
		case 3:
			testDropLeft(t, p, rand.Intn(20))
		case 4:
			testDropRight(t, p, rand.Intn(20))
		}

		if i == 1000 {
			testReset(t, p, true, 0)
		}
		if i == 2000 {
			testReset(t, p, false, 0)
		}
		if i%432 == 0 {
			testNormalizeReset(t, p, rand.Intn(20))
		}
		if i == 3000 {
			testNormalizeReset(t, p, 2+cap(p.buf))
		}
		if i == 4000 {
			testReset(t, p, false, 55)
		}
	}

	testExtendLeft(t, p, 0)
	testExtendRight(t, p, 0)
	testAppend(t, p, 0)
	testDropLeft(t, p, 0)
	testDropRight(t, p, 0)
}

func testAlloc(t *testing.T, p *PDU) *PDU {
	headerCap := rand.Int() % 256
	dataLen := rand.Int() % 256
	dataCap := dataLen + rand.Int()%256

	if p == nil {
		p = Alloc(headerCap, dataLen, dataCap)
	} else {
		origCap := cap(p.buf)
		realloc := p.Realloc(headerCap, dataLen, dataCap)
		reallocReal := cap(p.buf) != origCap
		check(t, reallocReal == realloc, "Wrong realloc result")
	}

	check(t, p.LeftCap() == headerCap, "LeftCap not headerCap")
	check(t, p.RightCap() >= dataCap, "RightCap not greater or equal to dataCap")
	check(t, p.Len() == dataLen, "Len not dataLen")
	checkCapLen(t, p)

	return p
}

func TestBasic(t *testing.T) {
	p := testAlloc(t, nil)

	for i := 0; i < 100; i++ {
		testPDU(t, p)
		testAlloc(t, p)
	}

	p = Alloc(5, 5, 5)
	testAlloc(t, p)
	testPDU(t, p)
}

func TestString(t *testing.T) {
	p := Alloc(0, 2, 2)
	p.Buf()[0] = 0xde
	p.Buf()[1] = 0xad
	check(t, p.String() == "dead", "String function did not work")
}

func TestAssert(t *testing.T) {
	assert(true, "Works great")
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	assert(false, "Assert failed")
}

func TestState(t *testing.T) {
	p := Alloc(0, 2, 2)
	p.SetState(0, 1)
	p.SetState(1, 2)
	p.SetState(2, 0)
	p.SetState(0, 6)

	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	p.SetState(5, 6)
}
