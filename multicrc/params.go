package multicrc

var (
	//CrcNone describes an empty CRC
	CrcNone = &Params{
		Len:  0,
		Name: "CrcNone",
	}

	//Crc16A describes a 16 bit long CRC named 'A'
	Crc16A = &Params{
		Len:           16,
		Name:          "Crc16A",
		Polynomial:    0x1021,
		InitialValue:  0xC6C6,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16ARC describes a 16 bit long CRC named 'ARC'
	Crc16ARC = &Params{
		Len:           16,
		Name:          "Crc16ARC",
		Polynomial:    0x8005,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16AUGCCITT describes a 16 bit long CRC named 'AUG_CCITT'
	Crc16AUGCCITT = &Params{
		Len:           16,
		Name:          "Crc16AUG_CCITT",
		Polynomial:    0x1021,
		InitialValue:  0x1D0F,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16BUYPASS describes a 16 bit long CRC named 'BUYPASS'
	Crc16BUYPASS = &Params{
		Len:           16,
		Name:          "Crc16BUYPASS",
		Polynomial:    0x8005,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16CCITTFALSE describes a 16 bit long CRC named 'CCITT_FALSE'
	Crc16CCITTFALSE = &Params{
		Len:           16,
		Name:          "Crc16CCITT_FALSE",
		Polynomial:    0x1021,
		InitialValue:  0xFFFF,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16CCITZERO describes a 16 bit long CRC named 'CCIT_ZERO'
	Crc16CCITZERO = &Params{
		Len:           16,
		Name:          "Crc16CCIT_ZERO",
		Polynomial:    0x1021,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16CDMA2000 describes a 16 bit long CRC named 'CDMA2000'
	Crc16CDMA2000 = &Params{
		Len:           16,
		Name:          "Crc16CDMA2000",
		Polynomial:    0xC867,
		InitialValue:  0xFFFF,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16DDS110 describes a 16 bit long CRC named 'DDS_110'
	Crc16DDS110 = &Params{
		Len:           16,
		Name:          "Crc16DDS_110",
		Polynomial:    0x8005,
		InitialValue:  0x800D,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16DECTR describes a 16 bit long CRC named 'DECT_R'
	Crc16DECTR = &Params{
		Len:           16,
		Name:          "Crc16DECT_R",
		Polynomial:    0x0589,
		InitialValue:  0x0000,
		FinalXOR:      0x0001,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16DECTX describes a 16 bit long CRC named 'DECT_X'
	Crc16DECTX = &Params{
		Len:           16,
		Name:          "Crc16DECT_X",
		Polynomial:    0x0589,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16DNP describes a 16 bit long CRC named 'DNP'
	Crc16DNP = &Params{
		Len:           16,
		Name:          "Crc16DNP",
		Polynomial:    0x3D65,
		InitialValue:  0x0000,
		FinalXOR:      0xFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16EN13757 describes a 16 bit long CRC named 'EN_13757'
	Crc16EN13757 = &Params{
		Len:           16,
		Name:          "Crc16EN_13757",
		Polynomial:    0x3D65,
		InitialValue:  0x0000,
		FinalXOR:      0xFFFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16GENIBUS describes a 16 bit long CRC named 'GENIBUS'
	Crc16GENIBUS = &Params{
		Len:           16,
		Name:          "Crc16GENIBUS",
		Polynomial:    0x1021,
		InitialValue:  0xFFFF,
		FinalXOR:      0xFFFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16KERMIT describes a 16 bit long CRC named 'KERMIT'
	Crc16KERMIT = &Params{
		Len:           16,
		Name:          "Crc16KERMIT",
		Polynomial:    0x1021,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16MAXIM describes a 16 bit long CRC named 'MAXIM'
	Crc16MAXIM = &Params{
		Len:           16,
		Name:          "Crc16MAXIM",
		Polynomial:    0x8005,
		InitialValue:  0x0000,
		FinalXOR:      0xFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16MCRF4XX describes a 16 bit long CRC named 'MCRF4XX'
	Crc16MCRF4XX = &Params{
		Len:           16,
		Name:          "Crc16MCRF4XX",
		Polynomial:    0x1021,
		InitialValue:  0xFFFF,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16MODBUS describes a 16 bit long CRC named 'MODBUS'
	Crc16MODBUS = &Params{
		Len:           16,
		Name:          "Crc16MODBUS",
		Polynomial:    0x8005,
		InitialValue:  0xFFFF,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16RIELLO describes a 16 bit long CRC named 'RIELLO'
	Crc16RIELLO = &Params{
		Len:           16,
		Name:          "Crc16RIELLO",
		Polynomial:    0x1021,
		InitialValue:  0xB2AA,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16T10DIF describes a 16 bit long CRC named 'T10_DIF'
	Crc16T10DIF = &Params{
		Len:           16,
		Name:          "Crc16T10_DIF",
		Polynomial:    0x8BB7,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16TELEDISK describes a 16 bit long CRC named 'TELEDISK'
	Crc16TELEDISK = &Params{
		Len:           16,
		Name:          "Crc16TELEDISK",
		Polynomial:    0xA097,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc16TMS37157 describes a 16 bit long CRC named 'TMS37157'
	Crc16TMS37157 = &Params{
		Len:           16,
		Name:          "Crc16TMS37157",
		Polynomial:    0x1021,
		InitialValue:  0x89EC,
		FinalXOR:      0x0000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16USB describes a 16 bit long CRC named 'USB'
	Crc16USB = &Params{
		Len:           16,
		Name:          "Crc16USB",
		Polynomial:    0x8005,
		InitialValue:  0xFFFF,
		FinalXOR:      0xFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16X25 describes a 16 bit long CRC named 'X_25'
	Crc16X25 = &Params{
		Len:           16,
		Name:          "Crc16X_25",
		Polynomial:    0x1021,
		InitialValue:  0xFFFF,
		FinalXOR:      0xFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc16XMODEM describes a 16 bit long CRC named 'XMODEM'
	Crc16XMODEM = &Params{
		Len:           16,
		Name:          "Crc16XMODEM",
		Polynomial:    0x1021,
		InitialValue:  0x0000,
		FinalXOR:      0x0000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc32BZIP2 describes a 32 bit long CRC named 'BZIP2'
	Crc32BZIP2 = &Params{
		Len:           32,
		Name:          "Crc32BZIP2",
		Polynomial:    0x04C11DB7,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0xFFFFFFFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc32C describes a 32 bit long CRC named 'C'
	Crc32C = &Params{
		Len:           32,
		Name:          "Crc32C",
		Polynomial:    0x1EDC6F41,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0xFFFFFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc32D describes a 32 bit long CRC named 'D'
	Crc32D = &Params{
		Len:           32,
		Name:          "Crc32D",
		Polynomial:    0xA833982B,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0xFFFFFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc32 describes a 32 bit long CRC
	Crc32 = &Params{
		Len:           32,
		Name:          "Crc32",
		Polynomial:    0x04C11DB7,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0xFFFFFFFF,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc32JAMCRC describes a 32 bit long CRC named 'JAMCRC'
	Crc32JAMCRC = &Params{
		Len:           32,
		Name:          "Crc32JAMCRC",
		Polynomial:    0x04C11DB7,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0x00000000,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc32MPEG2 describes a 32 bit long CRC named 'MPEG2'
	Crc32MPEG2 = &Params{
		Len:           32,
		Name:          "Crc32MPEG2",
		Polynomial:    0x04C11DB7,
		InitialValue:  0xFFFFFFFF,
		FinalXOR:      0x00000000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc32POSIX describes a 32 bit long CRC named 'POSIX'
	Crc32POSIX = &Params{
		Len:           32,
		Name:          "Crc32POSIX",
		Polynomial:    0x04C11DB7,
		InitialValue:  0x00000000,
		FinalXOR:      0xFFFFFFFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc32Q describes a 32 bit long CRC named 'Q'
	Crc32Q = &Params{
		Len:           32,
		Name:          "Crc32Q",
		Polynomial:    0x814141AB,
		InitialValue:  0x00000000,
		FinalXOR:      0x00000000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc32XFER describes a 32 bit long CRC named 'XFER'
	Crc32XFER = &Params{
		Len:           32,
		Name:          "Crc32XFER",
		Polynomial:    0x000000AF,
		InitialValue:  0x00000000,
		FinalXOR:      0x00000000,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc88H2F describes a 8 bit long CRC named '8H2F'
	Crc88H2F = &Params{
		Len:           8,
		Name:          "Crc88H2F",
		Polynomial:    0x2F,
		InitialValue:  0xFF,
		FinalXOR:      0xFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8CDMA2000 describes a 8 bit long CRC named 'CDMA2000'
	Crc8CDMA2000 = &Params{
		Len:           8,
		Name:          "Crc8CDMA2000",
		Polynomial:    0x9B,
		InitialValue:  0xFF,
		FinalXOR:      0x00,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8DARC describes a 8 bit long CRC named 'DARC'
	Crc8DARC = &Params{
		Len:           8,
		Name:          "Crc8DARC",
		Polynomial:    0x39,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc8DVBS2 describes a 8 bit long CRC named 'DVB_S2'
	Crc8DVBS2 = &Params{
		Len:           8,
		Name:          "Crc8DVB_S2",
		Polynomial:    0xD5,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8EBU describes a 8 bit long CRC named 'EBU'
	Crc8EBU = &Params{
		Len:           8,
		Name:          "Crc8EBU",
		Polynomial:    0x1D,
		InitialValue:  0xFF,
		FinalXOR:      0x00,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc8 describes a 8 bit long CRC
	Crc8 = &Params{
		Len:           8,
		Name:          "Crc8",
		Polynomial:    0x07,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8ICODE describes a 8 bit long CRC named 'ICODE'
	Crc8ICODE = &Params{
		Len:           8,
		Name:          "Crc8ICODE",
		Polynomial:    0x1D,
		InitialValue:  0xFD,
		FinalXOR:      0x00,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8ITU describes a 8 bit long CRC named 'ITU'
	Crc8ITU = &Params{
		Len:           8,
		Name:          "Crc8ITU",
		Polynomial:    0x07,
		InitialValue:  0x00,
		FinalXOR:      0x55,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8MAXIM describes a 8 bit long CRC named 'MAXIM'
	Crc8MAXIM = &Params{
		Len:           8,
		Name:          "Crc8MAXIM",
		Polynomial:    0x31,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc8ROHC describes a 8 bit long CRC named 'ROHC'
	Crc8ROHC = &Params{
		Len:           8,
		Name:          "Crc8ROHC",
		Polynomial:    0x07,
		InitialValue:  0xFF,
		FinalXOR:      0x00,
		ReflectInput:  true,
		ReflectOutput: true,
	}

	//Crc8SAEJ1850 describes a 8 bit long CRC named 'SAE_J1850'
	Crc8SAEJ1850 = &Params{
		Len:           8,
		Name:          "Crc8SAE_J1850",
		Polynomial:    0x1D,
		InitialValue:  0xFF,
		FinalXOR:      0xFF,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8SAEJ1850ZERO describes a 8 bit long CRC named 'SAE_J1850_ZERO'
	Crc8SAEJ1850ZERO = &Params{
		Len:           8,
		Name:          "Crc8SAE_J1850_ZERO",
		Polynomial:    0x1D,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  false,
		ReflectOutput: false,
	}

	//Crc8WCDMA describes a 8 bit long CRC named 'WCDMA'
	Crc8WCDMA = &Params{
		Len:           8,
		Name:          "Crc8WCDMA",
		Polynomial:    0x9B,
		InitialValue:  0x00,
		FinalXOR:      0x00,
		ReflectInput:  true,
		ReflectOutput: true,
	}
)
