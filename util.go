package firmata

const SevenBitMask byte = 0b01111111

func TwoByteToByte(a, b byte) byte {
	return (a & SevenBitMask) | ((b & SevenBitMask) << 7)
}

func TwoByteString(bytes []byte) string {
	if len(bytes)%2 == 1 {
		bytes = append(bytes, 0)
	}

	var s string
	for i := 0; i < len(bytes); i += 2 {
		s += string(TwoByteToByte(bytes[i], bytes[i+1]))
	}
	return s
}

func TwoByteRepresentationToByteSlice(bytes []byte) []byte {
	if len(bytes)%2 == 1 {
		bytes = append(bytes, 0)
	}

	d := make([]byte, len(bytes)/2)
	i := 0
	for di := range d {
		d[di] = TwoByteToByte(bytes[i], bytes[i+1])
		i += 2
	}
	return d
}

func ByteToTwoByte(b byte) (lsb, msb byte) {
	return b & SevenBitMask, (b >> 7) & SevenBitMask
}

func ByteSliceToTwoByteRepresentation(bytes []byte) []byte {
	d := make([]byte, len(bytes)*2)
	i := 0
	for _, b := range bytes {
		d[i], d[i+1] = ByteToTwoByte(b)
		i += 2
	}
	return d
}
