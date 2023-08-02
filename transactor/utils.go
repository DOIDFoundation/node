package transactor

func ValidateDoidName(s string, valid int) bool {
	length := getStringLength(s)
	if length > valid {
		return false
	} else {
		return true
	}
}

func getStringLength(s string) int {
	var strLength int
	i := 0
	b := []byte(s)
	for strLength = 0; i < len(b); strLength++ {
		if b[i] < 0x80 {
			i++ // ascii code : 1byte
		} else {
			// utf-8 dynamic length
			strLength++
			if b[i] < 0xE0 {
				i += 2
			} else if b[i] < 0xF0 {
				i += 3
			} else if b[i] < 0xF8 {
				i += 4
			} else if b[i] < 0xFC {
				i += 5
			} else {
				i += 6
			}
		}
	}
	return strLength
}
