package gpio

const gpioGetChipinfoIoctl uintptr = 0x8044b401
const gpioGetLineinfoIoctl uintptr = 0xc048b402
const gpioGetLinehandleIoctl uintptr = 0xc16cb403
const gpioGetLineeventIoctl uintptr = 0xc030b404
const gpiohandleGetLineValuesIoctl uintptr = 0xc040b408
const gpiohandleSetLineValuesIoctl uintptr = 0xc040b409

type LineFlag uint32

const LineKernel LineFlag = 0x00000001
const LineIsOut LineFlag = 0x00000002
const LineActiveLow LineFlag = 0x00000004
const LineOpenDrain LineFlag = 0x00000008
const LineOpenSource LineFlag = 0x00000010

type RequestFlag uint32

const RequestInput RequestFlag = 0x00000001
const RequestOutput RequestFlag = 0x00000002
const RequestActiveLow RequestFlag = 0x00000004
const RequestOpenDrain RequestFlag = 0x00000008
const RequestOpenSource RequestFlag = 0x00000010

type EventFlag uint32

const EventRisingEdge EventFlag = 0x00000001
const EventFallingEdge EventFlag = 0x00000002
