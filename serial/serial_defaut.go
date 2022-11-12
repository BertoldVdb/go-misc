//go:build !linux

package serial

func openPortOs(options *PortOptions) (Port, error) {
	return nil, errors.New("operating system is not supported")
}
